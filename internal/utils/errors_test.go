package utils

import (
	"strings"
	"testing"
)

func TestAxenError(t *testing.T) {
	err := NewAxenError("test message", "TEST_CODE")
	expected := "TEST_CODE: test message"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestNewManifestError(t *testing.T) {
	err := NewManifestError("manifest invalid")
	if err.Code != "MANIFEST_ERROR" {
		t.Errorf("Expected MANIFEST_ERROR, got %s", err.Code)
	}
	if err.Message != "manifest invalid" {
		t.Errorf("Expected 'manifest invalid', got %s", err.Message)
	}
}

func TestNewFrontmatterError(t *testing.T) {
	err := NewFrontmatterError("invalid yaml", "/path/to/skill")
	if err.Code != "FRONTMATTER_ERROR" {
		t.Errorf("Expected FRONTMATTER_ERROR, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "invalid yaml (at /path/to/skill)") {
		t.Errorf("Unexpected message: %s", err.Message)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("skill1", "old_ns", "new_ns")
	if err.Code != "CONFLICT_ERROR" {
		t.Errorf("Expected CONFLICT_ERROR, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "already installed") {
		t.Errorf("Unexpected message: %s", err.Message)
	}
}

func TestNewSourceError(t *testing.T) {
	err := NewSourceError("fetch failed", "http://source")
	if err.Code != "SOURCE_ERROR" {
		t.Errorf("Expected SOURCE_ERROR, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "fetch failed (source: http://source)") {
		t.Errorf("Unexpected message: %s", err.Message)
	}
}

func TestNewFileSystemError(t *testing.T) {
	err := NewFileSystemError("read failed", "/path")
	if err.Code != "FS_ERROR" {
		t.Errorf("Expected FS_ERROR, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "read failed (path: /path)") {
		t.Errorf("Unexpected message: %s", err.Message)
	}
}
