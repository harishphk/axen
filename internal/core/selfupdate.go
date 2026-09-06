package core

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SelfUpdateResult holds the outcome of a self-upgrade attempt.
type SelfUpdateResult struct {
	AlreadyUpToDate bool
	IsDevBuild      bool
	CurrentVersion  string
	NewVersion      string
	ExecutablePath  string
}

// UpgradeSelf performs a native in-process upgrade of the running binary.
// It uses only the Go standard library to check versions, download release archives,
// verify SHA-256 checksums, and atomically replace the current executable.
func UpgradeSelf(ctx context.Context, force bool, onProgress func(string)) (*SelfUpdateResult, error) {
	currentVersion := Version

	// 1. Resolve latest release tag
	latestTag, err := FetchLatestCliReleaseTag(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// 2. Safeguard for local development builds: do not overwrite without --force
	if currentVersion == "dev" && !force {
		return &SelfUpdateResult{
			IsDevBuild:     true,
			CurrentVersion: currentVersion,
			NewVersion:     latestTag,
		}, nil
	}

	// 3. Check if already up to date
	if !force && !IsNewerVersion(latestTag, currentVersion) {
		return &SelfUpdateResult{
			AlreadyUpToDate: true,
			CurrentVersion:  currentVersion,
			NewVersion:      latestTag,
		}, nil
	}

	// 4. Locate current executable and check for Homebrew installation
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to find current executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlink for %s: %w", exePath, err)
	}
	if strings.Contains(exePath, "/Cellar/") || strings.Contains(exePath, "/homebrew/") {
		return nil, fmt.Errorf("axen was installed via Homebrew (%s)\nPlease upgrade using: brew upgrade axen", exePath)
	}

	// 5. Build URLs and file metadata
	versionNum := strings.TrimPrefix(latestTag, "v")
	osName := runtime.GOOS
	archName := runtime.GOARCH

	ext := ".tar.gz"
	binaryName := "axen"
	if osName == "windows" {
		ext = ".zip"
		binaryName = "axen.exe"
	}
	fileName := fmt.Sprintf("axen_%s_%s_%s%s", versionNum, osName, archName, ext)
	downloadURL := fmt.Sprintf("https://github.com/harishphk/axen/releases/download/%s/%s", latestTag, fileName)
	checksumURL := fmt.Sprintf("https://github.com/harishphk/axen/releases/download/%s/checksums.txt", latestTag)

	client := &http.Client{Timeout: 60 * time.Second}

	// 6. Fetch expected checksum
	expectedChecksum, err := fetchExpectedChecksum(ctx, client, checksumURL, fileName)
	if err != nil {
		return nil, err
	}

	// 7. Download release archive
	if onProgress != nil {
		onProgress(fmt.Sprintf("Downloading Axen %s (%s/%s)...", latestTag, osName, archName))
	}
	archiveBytes, err := downloadFileBytes(ctx, client, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download release %s: %w", fileName, err)
	}

	// 8. Verify checksum
	if onProgress != nil {
		onProgress("Verifying SHA-256 checksum...")
	}
	if err := verifyChecksum(archiveBytes, expectedChecksum); err != nil {
		return nil, fmt.Errorf("checksum verification failed for %s: %w", fileName, err)
	}

	// 9. Extract binary
	newBinary, err := extractBinary(archiveBytes, osName, binaryName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract binary from release: %w", err)
	}

	// 10. Replace current executable
	if err := replaceExecutable(exePath, newBinary); err != nil {
		return nil, err
	}

	return &SelfUpdateResult{
		AlreadyUpToDate: false,
		CurrentVersion:  currentVersion,
		NewVersion:      latestTag,
		ExecutablePath:  exePath,
	}, nil
}

// fetchExpectedChecksum downloads checksums.txt and finds the SHA-256 for the target file.
func fetchExpectedChecksum(ctx context.Context, client *http.Client, checksumURL, targetFileName string) (string, error) {
	data, err := downloadFileBytes(ctx, client, checksumURL)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == targetFileName {
			return strings.ToLower(strings.TrimSpace(parts[0])), nil
		}
	}
	return "", fmt.Errorf("release archive %s not found in checksums.txt", targetFileName)
}

// downloadFileBytes performs an HTTP GET with context and returns response body bytes.
func downloadFileBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "axen-cli/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// verifyChecksum computes the SHA-256 hash of data and compares it against expectedHex.
func verifyChecksum(data []byte, expectedHex string) error {
	h := sha256.Sum256(data)
	actualHex := hex.EncodeToString(h[:])
	if actualHex != strings.ToLower(strings.TrimSpace(expectedHex)) {
		return fmt.Errorf("checksum mismatch (expected %s, got %s)", expectedHex, actualHex)
	}
	return nil
}

// extractBinary unpacks the target binary from a .tar.gz (Unix) or .zip (Windows) archive.
func extractBinary(archiveBytes []byte, osName, binaryName string) ([]byte, error) {
	if osName == "windows" {
		return extractFromZip(archiveBytes, binaryName)
	}
	return extractFromTarGz(archiveBytes, binaryName)
}

func extractFromZip(archiveBytes []byte, binaryName string) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse zip archive: %w", err)
	}
	for _, f := range zipReader.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("binary %s not found in zip archive", binaryName)
}

func extractFromTarGz(archiveBytes []byte, binaryName string) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(archiveBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip stream: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed reading tar archive: %w", err)
		}
		if filepath.Base(header.Name) == binaryName {
			return io.ReadAll(tarReader)
		}
	}
	return nil, fmt.Errorf("binary %s not found in tar archive", binaryName)
}

// replaceExecutable writes newBinary to a temporary file in the same directory and replaces exePath.
func replaceExecutable(exePath string, newBinary []byte) error {
	dir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(dir, ".axen-upgrade-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing to %s (try running with elevated permissions)", dir)
		}
		return fmt.Errorf("failed to create temporary update file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(newBinary); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write binary to temporary file: %w", err)
	}

	if err := tmpFile.Chmod(0755); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}
	_ = tmpFile.Close()

	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(exePath, oldPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to rename running executable: %w", err)
		}
		if err := os.Rename(tmpPath, exePath); err != nil {
			_ = os.Rename(oldPath, exePath) // rollback
			return fmt.Errorf("failed to install new executable: %w", err)
		}
		_ = os.Remove(oldPath)
		return nil
	}

	// On Unix (Linux & macOS), atomic rename replaces the directory entry
	if err := os.Rename(tmpPath, exePath); err != nil {
		_ = os.Remove(tmpPath)
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing to %s (try running with elevated permissions)", exePath)
		}
		return fmt.Errorf("failed to replace binary at %s: %w", exePath, err)
	}

	return nil
}
