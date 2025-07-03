// File: core/v2.go

// Package core contains the stable, versioned logic for the BTXZ archive format.
// This file implements the v2 specification.
// Core Version: v2
package core

import (
	"archive/zip"
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

	"github.com/klauspost/compress/zstd"
	"golang.org/x/crypto/argon2"
)

// --- v2 Core Constants & Header Definition ---

const (
	// coreVersionV2 is the integer identifier for this version of the format.
	coreVersionV2 = 2

	// Compression levels for Zstandard.
	levelFast    = uint8(0x01)
	levelDefault = uint8(0x02)
	levelBest    = uint8(0x03)
)

// BtxzHeaderV2 defines the binary structure of the v2 archive header.
// It is streamlined because encryption is now mandatory.
type BtxzHeaderV2 struct {
	Signature        [4]byte // "BTXZ"
	Version          uint16  // 2
	CompressionLevel uint8
	Salt             [saltSize]byte
	Argon2Time       uint32
	Argon2Memory     uint32
	Argon2Threads    uint8
	Nonce            [nonceSize]byte
}

// CreateArchiveV2 creates a new archive using the v2 format (ZIP -> ZSTD -> AES-GCM).
func CreateArchiveV2(archivePath string, inputPaths []string, password string, level string) error {
	if len(inputPaths) == 0 {
		return errors.New("no input files or folders specified")
	}
	if password == "" {
		return errors.New("a password is required for v2 archives")
	}

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("could not create archive file: %w", err)
	}
	defer archiveFile.Close()

	// 1. Configure Header
	var compLevel uint8
	var zstdLevel zstd.EncoderLevel
	switch level {
	case "fast":
		compLevel = levelFast
		zstdLevel = zstd.SpeedFastest
	case "best":
		compLevel = levelBest
		zstdLevel = zstd.SpeedBestCompression
	default: // "default"
		compLevel = levelDefault
		zstdLevel = zstd.SpeedDefault
	}

	header := BtxzHeaderV2{
		Signature:        [4]byte{'B', 'T', 'X', 'Z'},
		Version:          coreVersionV2,
		CompressionLevel: compLevel,
		Argon2Time:       argon2Time,
		Argon2Memory:     argon2Memory,
		Argon2Threads:    argon2Threads,
	}
	if _, err := rand.Read(header.Salt[:]); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	if _, err := rand.Read(header.Nonce[:]); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	key := argon2.IDKey([]byte(password), header.Salt[:], header.Argon2Time, header.Argon2Memory, header.Argon2Threads, argon2KeyLength)

	// 2. Prepare ZIP and ZSTD writers to stream data into an in-memory buffer.
	// Flow: Files -> ZIP (Store) -> ZSTD -> Buffer
	compressedBuffer := new(bytes.Buffer)
	zstdWriter, err := zstd.NewWriter(compressedBuffer, zstd.WithEncoderLevel(zstdLevel))
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}
	zipWriter := zip.NewWriter(zstdWriter)

	// 3. Walk through input paths and add files to the ZIP stream.
	for _, path := range inputPaths {
		err := addPathToZip(zipWriter, path)
		if err != nil {
			zipWriter.Close()
			zstdWriter.Close()
			return fmt.Errorf("failed while processing path %s: %w", path, err)
		}
	}
	zipWriter.Close()
	zstdWriter.Close()

	// 4. Write the final header to the archive file.
	if err := binary.Write(archiveFile, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to write archive header: %w", err)
	}

	// 5. Encrypt and write the compressed payload to the file.
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	encryptedPayload := gcm.Seal(nil, header.Nonce[:], compressedBuffer.Bytes(), nil)
	_, err = archiveFile.Write(encryptedPayload)

	return err
}

// addPathToZip is a helper function to recursively add a file or directory to a zip.Writer.
func addPathToZip(zw *zip.Writer, path string) error {
	basePath := filepath.Dir(path)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		basePath = path
	}

	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // Directories are created implicitly by their files.
		}

		relPath, err := filepath.Rel(basePath, filePath)
		if err != nil {
			return err
		}
		// Use forward slashes for cross-platform compatibility inside the archive.
		zipPath := filepath.ToSlash(relPath)

		// Create a zip FileHeader
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = zipPath
		// IMPORTANT: Use Store method to disable zip's own compression.
		// Zstandard will handle the compression for the entire stream.
		fh.Method = zip.Store

		writer, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
}

// getDecryptedReaderV2 opens a v2 archive, handles decryption, and returns a reader for the compressed payload.
func getDecryptedReaderV2(archivePath string, password string) (io.Reader, error) {
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer archiveFile.Close()

	var header BtxzHeaderV2
	if err := binary.Read(archiveFile, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read v2 archive header: %w", err)
	}

	key := argon2.IDKey([]byte(password), header.Salt[:], header.Argon2Time, header.Argon2Memory, header.Argon2Threads, argon2KeyLength)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	encryptedPayload, err := io.ReadAll(archiveFile)
	if err != nil {
		return nil, fmt.Errorf("could not read encrypted payload: %w", err)
	}

	decryptedPayload, err := gcm.Open(nil, header.Nonce[:], encryptedPayload, nil)
	if err != nil {
		return nil, errors.New("decryption failed: incorrect password or tampered archive")
	}

	return bytes.NewReader(decryptedPayload), nil
}

// ExtractArchiveV2 reads a v2 archive and extracts its contents.
func ExtractArchiveV2(archivePath, outputDir, password string) ([]string, error) {
	var skippedFiles []string

	payloadReader, err := getDecryptedReaderV2(archivePath, password)
	if err != nil {
		return nil, err
	}

	// Decompress the entire payload in memory first.
	// This is necessary because zip.NewReader needs an io.ReaderAt, which a streaming
	// zstd decompressor does not provide.
	zstdReader, err := zstd.NewReader(payloadReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()
	
	unzippedData, err := io.ReadAll(zstdReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress archive data: %w", err)
	}

	// Now read the decompressed (but still zipped) data.
	zipArchive, err := zip.NewReader(bytes.NewReader(unzippedData), int64(len(unzippedData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip stream from decompressed data: %w", err)
	}

	cleanOutputDir, err := filepath.Abs(filepath.Clean(outputDir))
	if err != nil {
		return nil, fmt.Errorf("could not resolve output directory path: %w", err)
	}

	for _, file := range zipArchive.File {
		targetPath := filepath.Join(cleanOutputDir, file.Name)
		cleanTargetPath := filepath.Clean(targetPath)

		// SECURITY: Prevent path traversal attacks.
		if !strings.HasPrefix(cleanTargetPath, cleanOutputDir) {
			skippedFiles = append(skippedFiles, file.Name)
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return skippedFiles, err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return skippedFiles, err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return skippedFiles, err
		}

		_, err = io.Copy(outFile, rc)

		rc.Close()
		outFile.Close()

		if err != nil {
			return skippedFiles, err
		}
	}
	return skippedFiles, nil
}

// ListArchiveContentsV2 reads a v2 archive and lists its contents.
func ListArchiveContentsV2(archivePath, password string) ([]ArchiveEntry, error) {
	payloadReader, err := getDecryptedReaderV2(archivePath, password)
	if err != nil {
		return nil, err
	}

	zstdReader, err := zstd.NewReader(payloadReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	unzippedData, err := io.ReadAll(zstdReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress archive data: %w", err)
	}

	zipArchive, err := zip.NewReader(bytes.NewReader(unzippedData), int64(len(unzippedData)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip stream: %w", err)
	}

	var contents []ArchiveEntry
	for _, file := range zipArchive.File {
		entry := ArchiveEntry{
			Mode: file.Mode().String(),
			Size: int64(file.UncompressedSize64),
			Name: file.Name,
		}
		contents = append(contents, entry)
	}
	return contents, nil
}
