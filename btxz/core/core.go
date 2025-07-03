// File: core/core.go

package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// peekVersion opens an archive file, reads just the header to identify the
// format version, and then closes the file. This allows the dispatcher to
// call the correct version-specific logic.
func peekVersion(archivePath string) (uint16, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return 0, fmt.Errorf("could not open archive file: %w", err)
	}
	defer file.Close()

	// The header structure is designed so the signature (4 bytes) and version (2 bytes)
	// are always at the beginning. We read just enough to determine the version.
	headerStart := make([]byte, 6)
	if _, err := file.Read(headerStart); err != nil {
		return 0, fmt.Errorf("could not read archive header: %w", err)
	}

	// Check signature
	if string(headerStart[0:4]) != magicSignature {
		return 0, errors.New("not a valid BTXZ archive")
	}

	// Read version (Little Endian)
	version := binary.LittleEndian.Uint16(headerStart[4:6])
	return version, nil
}

// CreateArchive creates a new archive. By default, it creates the latest version (v2).
// It serves as the single entry point for archive creation.
func CreateArchive(archivePath string, inputPaths []string, password string, level string) error {
	// For now, all new archives are created using the v2 format.
	return CreateArchiveV2(archivePath, inputPaths, password, level)
}

// ExtractArchive inspects the archive version and calls the appropriate
// version-specific extraction function.
func ExtractArchive(archivePath, outputDir, password string) ([]string, error) {
	version, err := peekVersion(archivePath)
	if err != nil {
		return nil, err
	}

	switch version {
	case coreVersionV1:
		// Note: We are calling the function from the v1.go file.
		return ExtractArchiveV1(archivePath, outputDir, password)
	case coreVersionV2:
		// Note: We are calling the function from the v2.go file.
		return ExtractArchiveV2(archivePath, outputDir, password)
	default:
		return nil, fmt.Errorf("unsupported archive core version: v%d", version)
	}
}

// ListArchiveContents inspects the archive version and calls the appropriate
// version-specific listing function.
func ListArchiveContents(archivePath, password string) ([]ArchiveEntry, error) {
	version, err := peekVersion(archivePath)
	if err != nil {
		return nil, err
	}

	switch version {
	case coreVersionV1:
		return ListArchiveContentsV1(archivePath, password)
	case coreVersionV2:
		return ListArchiveContentsV2(archivePath, password)
	default:
		return nil, fmt.Errorf("unsupported archive core version: v%d", version)
	}
}
