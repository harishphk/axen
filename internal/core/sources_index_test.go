package core

import (
	"axen/internal/models"
	"os"
	"testing"
)

func TestSourcesIndex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	_ = os.Setenv("AXEN_TEST_HOME", tempDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	// Test writing
	cache := models.NewSourcesCache()
	cache.Namespaces["test-ns"] = models.CacheNamespace{
		Available: map[string]models.AvailableSkill{
			"test-skill": {Path: "test-skill", Version: "1.0"},
		},
	}
	err = WriteSourcesIndex(cache)
	if err != nil {
		t.Fatalf("Failed to write sources index: %v", err)
	}

	// Test reading
	readCache, err := ReadSourcesIndex()
	if err != nil {
		t.Fatalf("Failed to read sources index: %v", err)
	}
	if readCache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	if _, ok := readCache.Namespaces["test-ns"]; !ok {
		t.Errorf("Expected test-ns in namespaces")
	} else if _, ok := readCache.Namespaces["test-ns"].Available["test-skill"]; !ok {
		t.Errorf("Expected test-skill in test-ns available skills")
	}
}
