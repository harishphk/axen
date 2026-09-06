package core

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestUpgradeSelfAlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	origVersion := Version
	Version = "v999.0.0"
	defer func() { Version = origVersion }()

	res, err := UpgradeSelf(context.Background(), false, nil)
	if err != nil {
		t.Fatalf("expected no error when already up to date, got: %v", err)
	}
	if !res.AlreadyUpToDate {
		t.Errorf("expected AlreadyUpToDate to be true")
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho 'hello from mock binary'\n")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "axen",
		Mode: 0755,
		Size: int64(len(binaryContent)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(binaryContent); err != nil {
		t.Fatalf("failed to write binary content: %v", err)
	}
	_ = tw.Close()
	_ = gw.Close()

	archiveBytes := buf.Bytes()

	extracted, err := extractBinary(archiveBytes, "linux", "axen")
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Fatalf("extracted content mismatch: got %q, want %q", extracted, binaryContent)
	}

	// Test missing binary
	_, err = extractBinary(archiveBytes, "linux", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
}

func TestExtractBinaryZip(t *testing.T) {
	binaryContent := []byte("MZ_MOCK_WINDOWS_EXE")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	f, err := zw.Create("axen.exe")
	if err != nil {
		t.Fatalf("failed to create file in zip: %v", err)
	}
	if _, err := f.Write(binaryContent); err != nil {
		t.Fatalf("failed to write to zip: %v", err)
	}
	_ = zw.Close()

	archiveBytes := buf.Bytes()

	extracted, err := extractBinary(archiveBytes, "windows", "axen.exe")
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Fatalf("extracted zip content mismatch: got %q, want %q", extracted, binaryContent)
	}

	// Test missing binary
	_, err = extractBinary(archiveBytes, "windows", "other.exe")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("mock archive payload")
	h := sha256.Sum256(data)
	actualHash := hex.EncodeToString(h[:])

	// Valid hash (case insensitive)
	if err := verifyChecksum(data, strings.ToUpper(actualHash)); err != nil {
		t.Fatalf("expected valid checksum to succeed, got: %v", err)
	}

	// Invalid hash
	if err := verifyChecksum(data, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

func TestUpgradeSelfDevBuild(t *testing.T) {
	origVersion := Version
	Version = "dev"
	defer func() { Version = origVersion }()

	res, err := UpgradeSelf(context.Background(), false, nil)
	if err != nil {
		t.Fatalf("expected no error for dev build check, got: %v", err)
	}
	if !res.IsDevBuild {
		t.Errorf("expected IsDevBuild to be true")
	}
}
