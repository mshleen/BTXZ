// File: main.go

// Package main implements the command-line interface for BTXZ.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"btxz/core"
	"btxz/update"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const version = "0.0.0‑dev" // <-- this will be auto‑replaced by CI

const asciiArtLogo = `BTXZ`

// main is the entry point for the application. It sets up the command structure
// and runs a background check for new updates.
func main() {
	// Run the update check in a separate goroutine so it doesn't block the UI.
	go update.CheckForUpdates(version)

	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// NewRootCmd creates and configures the main 'btxz' command and its subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "btxz",
		Short:   "BTXZ: A secure and efficient file archiver.",
		Version: version,
		Long: `BTXZ is a professional command-line tool for creating and extracting
securely encrypted, highly compressed archives using a proprietary format.`,
		// Suppress the default 'completion' command from cobra.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Allow users to disable all styling for CI/CD or accessibility.
			if quiet, _ := cmd.Flags().GetBool("no-style"); quiet {
				pterm.DisableStyling()
				pterm.DisableColor()
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// After any command runs, display the update notification if one is available.
			update.DisplayUpdateNotification()
		},
	}

	rootCmd.SetVersionTemplate(`{{printf "btxz version %s\n" .Version}}`)
	rootCmd.Flags().Bool("no-style", false, "Disable all styling and colors")

	rootCmd.AddCommand(
		NewCreateCmd(),
		NewExtractCmd(),
		NewListCmd(),
		NewUpdateCmd(),
	)

	return rootCmd
}

// NewCreateCmd configures the 'create' command.
func NewCreateCmd() *cobra.Command {
	var (
		outputFile string
		password   string
		level      string
	)
	createCmd := &cobra.Command{
		Use:   "create [file/folder...]",
		Short: "Create a new secure archive",
		Long: `Packages one or more files and/or folders into a single compressed and encrypted .btxz archive.
New archives are created using the modern v2 format (Zstandard + AES-GCM).`,
		Example: `  btxz create ./doc.pdf ./images -o archive.btxz -p "s3cr3t!" --level fast`,
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printCommandHeader("Create Secure Archive (v2)")

			if outputFile == "" {
				handleCmdError("Output file path must be specified with -o or --output.")
			}
			// Validate compression level
			level = strings.ToLower(level)
			if level != "fast" && level != "default" && level != "best" {
				handleCmdError("Invalid level '%s'. Must be one of: fast, default, best.", level)
			}
			
			// A password is now required for creation.
			promptForPassword(&password)

			spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Processing %d input paths...", len(args)))
			err := core.CreateArchive(outputFile, args, password, level)
			spinner.Stop()

			if err != nil {
				handleCmdError("Failed to create archive: %v", err)
			}
			pterm.Success.Println("Archive creation complete.")
			pterm.DefaultBox.WithTitle("Summary").Println(
				fmt.Sprintf("Archive: %s\nEncrypted: %t\nCompression: %s",
					pterm.Green(outputFile), true, pterm.Cyan(level)),
			)
		},
	}
	createCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path for the new archive file (required)")
	createCmd.Flags().StringVarP(&password, "password", "p", "", "Password for encryption (prompts if empty, required)")
	createCmd.Flags().StringVarP(&level, "level", "l", "default", "Compression level (fast, default, best)")

	return createCmd
}

// NewExtractCmd configures the 'extract' command.
func NewExtractCmd() *cobra.Command {
	var (
		outputDir string
		password  string
	)
	extractCmd := &cobra.Command{
		Use:     "extract <archive.btxz>",
		Short:   "Extract files from an archive",
		Long:    `Decompresses and decrypts a .btxz archive into the specified directory. Automatically handles v1 and v2 formats.`,
		Example: `  btxz extract data.btxz -o ./restored_data`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			printCommandHeader("Extract BTXZ Archive")
			archivePath := args[0]
			
			// Prompt for password if not provided. Extraction might work on old v1 files without a password.
			if password == "" {
				pass, _ := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Enter decryption password (if required)")
				password = pass
			}

			spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Extracting '%s'...", filepath.Base(archivePath)))
			skippedFiles, err := core.ExtractArchive(archivePath, outputDir, password)
			spinner.Stop()

			if err != nil {
				if strings.Contains(err.Error(), "decryption failed") {
					handleCmdError("Decryption failed. Please check if your password is correct.")
				}
				handleCmdError("A fatal error occurred during extraction: %v", err)
			}

			if len(skippedFiles) > 0 {
				pterm.Warning.Println("Extraction completed with warnings.")
				var skippedList strings.Builder
				for _, file := range skippedFiles {
					skippedList.WriteString(fmt.Sprintf("  - %s\n", file))
				}
				pterm.DefaultBox.WithTitle(pterm.LightYellow("Skipped Unsafe Files")).WithBoxStyle(pterm.NewStyle(pterm.FgYellow)).Println(
					"The following files were not extracted to protect your system:\n" + skippedList.String(),
				)
			} else {
				pterm.Success.Println("Archive extraction complete.")
			}
		},
	}
	extractCmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Directory to extract files to")
	extractCmd.Flags().StringVarP(&password, "password", "p", "", "Password for decryption (prompts if empty)")
	return extractCmd
}

// NewListCmd configures the 'list' command.
func NewListCmd() *cobra.Command {
	var password string
	listCmd := &cobra.Command{
		Use:     "list <archive.btxz>",
		Short:   "List the contents of an archive",
		Long:    `Shows a list of files and folders inside a .btxz archive without extracting them. Automatically handles v1 and v2 formats.`,
		Example: `  btxz list my_archive.btxz -p "s3cr3t!"`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pterm.DefaultHeader.WithMargin(2).Println("List Archive Contents")
			archivePath := args[0]
			
			if password == "" {
				pass, _ := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Enter decryption password (if required)")
				password = pass
			}

			spinner, _ := pterm.DefaultSpinner.Start("Reading archive metadata...")
			contents, err := core.ListArchiveContents(archivePath, password)
			spinner.Stop()

			if err != nil {
				if strings.Contains(err.Error(), "decryption failed") {
					handleCmdError("Decryption failed. Cannot list contents without the correct password.")
				}
				handleCmdError("Failed to list archive contents: %v", err)
			}

			pterm.Success.Printf("Found %d entries in %s.\n", len(contents), filepath.Base(archivePath))
			tableData := pterm.TableData{{"Mode", "Size (bytes)", "Name"}}
			for _, item := range contents {
				tableData = append(tableData, []string{item.Mode, fmt.Sprintf("%d", item.Size), item.Name})
			}
			pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()
		},
	}
	listCmd.Flags().StringVarP(&password, "password", "p", "", "Password for decryption (prompts if empty)")
	return listCmd
}

// NewUpdateCmd configures the 'update' command.
func NewUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update btxz to the latest version",
		Long:  `Checks for the latest version on GitHub and performs an in-place update if available.`,
		Run: func(cmd *cobra.Command, args []string) {
			pterm.DefaultHeader.Println("Application Self-Update")
			if err := update.PerformUpdate(version); err != nil {
				handleCmdError("Update failed: %v", err)
			}
			pterm.Success.Println("BTXZ has been updated successfully!")
		},
	}
}

// --- Helper Functions ---

// handleCmdError prints a formatted error message and exits the application.
func handleCmdError(format string, a ...interface{}) {
	pterm.Error.Printf(format+"\n", a...)
	os.Exit(1)
}

// promptForPassword checks if a password string is empty and, if so, prompts
// the user for it. For v2 creation, an empty password after the prompt is a fatal error.
func promptForPassword(password *string) {
	if *password == "" {
		pterm.Info.Println("Password not provided via flag.")
		pass, _ := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Enter encryption password (required)")
		*password = pass
	}
	if *password == "" {
		handleCmdError("A password is required to create a secure archive.")
	}
}

// printCommandHeader displays the standard logo and title for a command.
func printCommandHeader(title string) {
	pterm.DefaultCenter.Println(pterm.DefaultBigText.WithLetters(pterm.NewLettersFromStringWithStyle(asciiArtLogo, pterm.NewStyle(pterm.FgCyan))).Srender())
	pterm.DefaultHeader.Println(title)
}
