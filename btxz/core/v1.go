// File: core/v1.go

// Package core contains the stable, versioned logic for the BTXZ archive format.
// This ensures that future updates to the tool can still read older archive
// versions by including their respective core files.
// This file implements the v1 specification.
// Core Version: v1
package core

import (
	"archive/tar"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"golang.org/x/crypto/argon2"
)

// --- v1 Core Constants & Header Definition ---

const (
	// magicSignature is a 4-byte identifier at the start of every BTXZ file.
	magicSignature = "BTXZ"
	// coreVersionV1 is the integer identifier for this version of the format.
	coreVersionV1 = 1

	// Protection modes for the archive payload.
	modeUnprotected = uint8(0x00)
	modeEncrypted   = uint8(0x01)

	// Filename encryption modes (in v1, this is tied to payload encryption).
	namesUnencrypted = uint8(0x00)
	namesEncrypted   = uint8(0x01)

	// Cryptographic parameters for Argon2id and AES-256-GCM.
	saltSize        = 16
	nonceSize       = 12 // Standard for GCM
	argon2KeyLength = 32 // 32 bytes = 256 bits for AES-256
	argon2Time      = 1
	argon2Memory    = 64 * 1024 // 64 MB
	argon2Threads   = 4
)

// BtxzHeaderV1 defines the binary structure of the v1 archive header.
// This data is written at the beginning of the archive file.
type BtxzHeaderV1 struct {
	Signature          [4]byte // Should always be "BTXZ"
	Version            uint16  // e.g., 1 for v1
	ProtectionMode     uint8   // 0x00 for none, 0x01 for AES-GCM
	FileNameEncryption uint8   // 0x00 for none, 0x01 for encrypted
	Salt               [saltSize]byte
	Argon2Time         uint32
	Argon2Memory       uint32
	Argon2Threads      uint8
	Nonce              [nonceSize]byte
}

// ArchiveEntry holds structured information about a single file within the archive,
// used primarily for the 'list' command.
type ArchiveEntry struct {
	Mode string
	Size int64
	Name string
}


func CreateArchive(archivePath string, inputPaths []string, password string, level string, encryptNames bool) error {
	if len(inputPaths) == 0 {
		return errors.New("no input files or folders specified")
	}
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("could not create archive file: %w", err)
	}
	defer archiveFile.Close()

	// 1. Configure Header based on password
	header := BtxzHeaderV1{
		Signature: [4]byte{'B', 'T', 'X', 'Z'},
		Version:   coreVersionV1,
	}
	var key []byte
	if password == "" {
		header.ProtectionMode = modeUnprotected
		header.FileNameEncryption = namesUnencrypted
	} else {
		header.ProtectionMode = modeEncrypted
		header.FileNameEncryption = namesEncrypted // In v1, names are always encrypted if a password is used
		header.Argon2Time = argon2Time
		header.Argon2Memory = argon2Memory
		header.Argon2Threads = argon2Threads
		if _, err := rand.Read(header.Salt[:]); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		if _, err := rand.Read(header.Nonce[:]); err != nil {
			return fmt.Errorf("failed to generate nonce: %w", err)
		}
		key = argon2.IDKey([]byte(password), header.Salt[:], header.Argon2Time, header.Argon2Memory, header.Argon2Threads, argon2KeyLength)
	}

	// 2. Write the final header to the archive file.
	if err := binary.Write(archiveFile, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to write archive header: %w", err)
	}

	// 3. Prepare TAR and XZ writers to stream data into an in-memory buffer.
	compressedBuffer := new(bytes.Buffer)
	xzWriter, err := xz.NewWriter(compressedBuffer)
	if err != nil {
		return fmt.Errorf("failed to create xz writer: %w", err)
	}
	tarWriter := tar.NewWriter(xzWriter)

	// 4. Walk through input paths and add files to the TAR stream.
	for _, path := range inputPaths {
		basePath := filepath.Dir(path)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("could not stat input path %s: %w", path, err)
		}
		if info.IsDir() {
			basePath = path
		}
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil // Directories are created implicitly by their files.
			}
			return addFileToTar(tarWriter, filePath, basePath)
		})
		if err != nil {
			tarWriter.Close()
			xzWriter.Close()
			return fmt.Errorf("failed while walking path %s: %w", path, err)
		}
	}
	tarWriter.Close()
	xzWriter.Close()

	// 5. Encrypt (if needed) and write the compressed payload to the file.
	if password != "" {
		block, _ := aes.NewCipher(key)
		gcm, _ := cipher.NewGCM(block)
		encryptedPayload := gcm.Seal(nil, header.Nonce[:], compressedBuffer.Bytes(), nil)
		_, err = archiveFile.Write(encryptedPayload)
	} else {
		_, err = io.Copy(archiveFile, compressedBuffer)
	}
	return err
}

// addFileToTar is a helper function to write a single file into a tar.Writer stream.
func addFileToTar(tw *tar.Writer, filePath, basePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}
	// Use relative paths within the archive for portability.
	header.Name, _ = filepath.Rel(basePath, filePath)
	// Use forward slashes for cross-platform compatibility.
	header.Name = filepath.ToSlash(header.Name)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, file); err != nil {
		return err
	}
	return nil
}

// getDecryptedReader opens an archive, validates its header, handles decryption,
// and returns a reader for the compressed payload (the TAR stream).
func getDecryptedReader(archivePath string, password string) (io.ReadCloser, error) {
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}

	var header BtxzHeaderV1
	if err := binary.Read(archiveFile, binary.LittleEndian, &header); err != nil {
		archiveFile.Close()
		return nil, fmt.Errorf("failed to read archive header: %w", err)
	}

	if string(header.Signature[:]) != magicSignature {
		archiveFile.Close()
		return nil, errors.New("not a valid BTXZ archive")
	}
	// This is the key for future-proofing: check the version.
	if header.Version != coreVersionV1 {
		archiveFile.Close()
		return nil, fmt.Errorf("unsupported archive core version: v%d. This tool supports v%d", header.Version, coreVersionV1)
	}

	// Handle unencrypted archives.
	if header.ProtectionMode == modeUnprotected {
		return archiveFile, nil
	}

	// Handle encrypted archives.
	if password == "" {
		archiveFile.Close()
		return nil, errors.New("archive is encrypted, but no password was provided")
	}

	key := argon2.IDKey([]byte(password), header.Salt[:], header.Argon2Time, header.Argon2Memory, header.Argon2Threads, argon2KeyLength)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	encryptedPayload, err := io.ReadAll(archiveFile)
	archiveFile.Close() // Close file immediately after reading.
	if err != nil {
		return nil, err
	}

	decryptedPayload, err := gcm.Open(nil, header.Nonce[:], encryptedPayload, nil)
	if err != nil {
		return nil, errors.New("decryption failed: incorrect password or tampered archive")
	}
	return io.NopCloser(bytes.NewReader(decryptedPayload)), nil
}

// ExtractArchive reads a v1 archive and extracts its contents to a specified directory.
// It is designed to be resilient, skipping unsafe files instead of halting.
// It returns a list of skipped file paths and a potential fatal error.
func ExtractArchive(archivePath, outputDir, password string) ([]string, error) {
	var skippedFiles []string
	payloadReader, err := getDecryptedReader(archivePath, password)
	if err != nil {
		return nil, err // Return immediately on fatal read/decryption errors.
	}
	defer payloadReader.Close()

	xzReader, err := xz.NewReader(payloadReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create xz reader: %w", err)
	}
	tarReader := tar.NewReader(xzReader)

	// Clean the output directory path once for reliable prefix checking.
	cleanOutputDir, err := filepath.Abs(filepath.Clean(outputDir))
	if err != nil {
		return nil, fmt.Errorf("could not resolve output directory path: %w", err)
	}

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive successfully reached.
		}
		if err != nil {
			// This indicates a corrupted tar stream.
			return skippedFiles, fmt.Errorf("error reading archive stream: %w", err)
		}

		// Construct the full, absolute path for the file to be extracted.
		targetPath := filepath.Join(cleanOutputDir, hdr.Name)
		cleanTargetPath := filepath.Clean(targetPath)

		// SECURITY: Prevent path traversal attacks (e.g., file paths like "../../../etc/passwd").
		// Check if the final cleaned path is still within the intended output directory.
		if !strings.HasPrefix(cleanTargetPath, cleanOutputDir) {
			skippedFiles = append(skippedFiles, hdr.Name)
			continue // Skip this unsafe file and continue to the next.
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return skippedFiles, err
			}
		case tar.TypeReg:
			// Ensure parent directory of the file exists.
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return skippedFiles, err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
			if err != nil {
				return skippedFiles, err
			}
			// Use defer in a closure to ensure file is closed even on copy error.
			func() {
				defer outFile.Close()
				_, err = io.Copy(outFile, tarReader)
			}()
			if err != nil {
				return skippedFiles, err
			}
		}
	}
	// Return the list of skipped files and a nil error to indicate overall success.
	return skippedFiles, nil
}

// ListArchiveContents reads an archive and returns a slice of ArchiveEntry structs
// representing the files within, without extracting them.
func ListArchiveContents(archivePath, password string) ([]ArchiveEntry, error) {
	payloadReader, err := getDecryptedReader(archivePath, password)
	if err != nil {
		return nil, err
	}
	defer payloadReader.Close()

	xzReader, err := xz.NewReader(payloadReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create xz reader: %w", err)
	}
	tarReader := tar.NewReader(xzReader)

	var contents []ArchiveEntry
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		entry := ArchiveEntry{
			Mode: os.FileMode(hdr.Mode).String(),
			Size: hdr.Size,
			Name: hdr.Name,
		}
		contents = append(contents, entry)
	}
	return contents, nil
}
