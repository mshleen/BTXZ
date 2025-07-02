// File: update.go
// This package handles the application's self-updating mechanism.

package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"github.com/inconshreveable/go-update"
	"github.com/pterm/pterm"
)
const versionURL = "https://raw.githubusercontent.com/BlackTechX011/BTXZ/main/version.json"

// updateArt is the visual warning for an available update.
const updateArt = `
      ▲
     / \
    / ! \
   /_____\
  UPDATE AVAILABLE`

// ReleaseInfo defines the structure of the version.json file on GitHub.
type ReleaseInfo struct {
	Version   string                     `json:"version"`
	Notes     string                     `json:"notes"`
	Platforms map[string]PlatformDetails `json:"platforms"`
}

// PlatformDetails contains the download URL for a specific OS/architecture.
type PlatformDetails struct {
	URL string `json:"url"`
}

// a an in-memory cache for the latest release info.
var (
	latestRelease *ReleaseInfo
	checkOnce     sync.Once
	mu            sync.RWMutex
)

// CheckForUpdates fetches release information from GitHub.
// It is designed to be run in a goroutine and will not block.
// It handles network errors gracefully by simply doing nothing.
func CheckForUpdates(currentVersion string) {
	checkOnce.Do(func() {
		resp, err := http.Get(versionURL)
		if err != nil {
			return // Fail silently on network errors
		}
		defer resp.Body.Close()

		var release ReleaseInfo
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return // Fail silently on JSON parsing errors
		}

		// Simple string comparison works for "v1.0" > "v1.0"
		if release.Version > currentVersion {
			mu.Lock()
			latestRelease = &release
			mu.Unlock()
		}
	})
}

// DisplayUpdateNotification prints a prominent warning if a new version is available.
func DisplayUpdateNotification() {
	mu.RLock()
	release := latestRelease
	mu.RUnlock()

	if release != nil {
		pterm.Println() // Add some space
		message := fmt.Sprintf("A new version (%s) is available!\n\nNotes: %s\n\n%s",
			pterm.LightGreen(release.Version),
			release.Notes,
			pterm.LightYellow("Run 'btxz update' to get the latest features and security fixes."),
		)
		pterm.DefaultBox.WithTitle(pterm.LightYellow("UPDATE AVAILABLE")).WithTitleTopCenter().WithBoxStyle(pterm.NewStyle(pterm.FgYellow)).Println(pterm.FgYellow.Sprint(updateArt) + "\n\n" + message)
	}
}

// PerformUpdate executes the self-update process.
func PerformUpdate(currentVersion string) error {
	pterm.Info.Println("Checking for the latest version...")
	// We run the check again here to ensure we have the absolute latest info.
	CheckForUpdates(currentVersion)

	mu.RLock()
	release := latestRelease
	mu.RUnlock()

	if release == nil {
		return errors.New("you are already running the latest version, or the update check failed")
	}

	platformKey := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	platformInfo, ok := release.Platforms[platformKey]
	if !ok {
		return fmt.Errorf("no update available for your platform: %s", platformKey)
	}

	pterm.Info.Printf("Downloading new version %s...\n", release.Version)
	resp, err := http.Get(platformInfo.URL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	pterm.Info.Println("Applying update... Your OS may ask for permissions.")
	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		// The library can return an error that we can check.
		if rerr := update.RollbackError(err); rerr != nil {
			return fmt.Errorf("failed to apply update and rollback failed: %v", rerr)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}
	return nil
}
