package sources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchLocal(t *testing.T) {
	tempDir := t.TempDir()

	localDir := filepath.Join(tempDir, "my_local_source")
	_ = os.MkdirAll(localDir, 0755)

	// Test existing directory
	localPath, ref, err := FetchLocal(localDir)
	if err != nil {
		t.Fatalf("FetchLocal failed: %v", err)
	}

	expectedPath, _ := filepath.Abs(localDir)
	if localPath != expectedPath {
		t.Fatalf("Expected localPath %s, got %s", expectedPath, localPath)
	}

	if !strings.HasPrefix(ref, "local-") {
		t.Fatalf("Expected ref to start with 'local-', got %s", ref)
	}

	// Test non-existent directory
	missingDir := filepath.Join(tempDir, "does_not_exist")
	_, _, err = FetchLocal(missingDir)
	if err == nil {
		t.Fatalf("Expected error for non-existent directory")
	}
	if !strings.Contains(err.Error(), "Local directory does not exist") {
		t.Fatalf("Expected directory does not exist error, got %v", err)
	}
}
