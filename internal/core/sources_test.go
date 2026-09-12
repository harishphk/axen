package core

import (
	"os"
	"testing"

	"github.com/harishphk/axen/internal/models"
)

func setupSourcesTestEnvironment(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	t.Cleanup(func() { _ = os.Unsetenv("AXEN_TEST_HOME") })

	lockfile := models.NewLockfile()
	_ = WriteLockfile(lockfile)

	return tmpDir
}

func TestSources_AddSource(t *testing.T) {
	setupSourcesTestEnvironment(t)

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "test-ns",
		Skills: map[string]models.SkillEntry{
			"skill-one": {
				Path:    "skills/one",
				Version: "1.0.0",
			},
		},
	}

	err := AddSource(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     "https://github.com/test/repo",
		SourceType:    "git",
		UpdatePolicy:  "daily",
		Manifest:      manifest,
	})
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	lockfile, err := ReadLockfile()
	if err != nil {
		t.Fatalf("ReadLockfile failed: %v", err)
	}
	entry, ok := lockfile.Namespaces["test-ns"]
	if !ok {
		t.Fatalf("expected namespace test-ns to be added")
	}
	if entry.Source != "https://github.com/test/repo" {
		t.Errorf("expected source https://github.com/test/repo, got %s", entry.Source)
	}
	if entry.UpdatePolicy != "daily" {
		t.Errorf("expected update policy daily, got %s", entry.UpdatePolicy)
	}

	// Verify sources cache
	cache, err := ReadSourcesIndex()
	if err != nil {
		t.Fatalf("ReadSourcesIndex failed: %v", err)
	}
	cacheNs, ok := cache.Namespaces["test-ns"]
	if !ok {
		t.Fatalf("expected cache for test-ns")
	}
	if skill, ok := cacheNs.Available["skill-one"]; !ok || skill.Version != "1.0.0" {
		t.Errorf("expected skill-one with version 1.0.0 in cache")
	}
}

func TestSources_SetSourcePolicy(t *testing.T) {
	setupSourcesTestEnvironment(t)

	err := AddSource(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     "https://github.com/test/repo",
		SourceType:    "git",
	})
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	err = SetSourcePolicy("test-ns", "weekly")
	if err != nil {
		t.Fatalf("SetSourcePolicy failed: %v", err)
	}

	lockfile, _ := ReadLockfile()
	if lockfile.Namespaces["test-ns"].UpdatePolicy != "weekly" {
		t.Errorf("expected update policy to be weekly, got %s", lockfile.Namespaces["test-ns"].UpdatePolicy)
	}

	// Test non-existent namespace
	err = SetSourcePolicy("non-existent", "daily")
	if err == nil {
		t.Errorf("expected error for non-existent namespace")
	}
}

func TestSources_AddSource_InitializesNilCache(t *testing.T) {
	setupSourcesTestEnvironment(t)

	// Explicitly remove sources index file to ensure cache is nil initially
	sourcesIndexPath := GetSourcesIndexPath()
	_ = os.Remove(sourcesIndexPath)

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "fresh-ns",
		Skills: map[string]models.SkillEntry{
			"fresh-skill": {Path: "skills/fresh", Version: "1.0"},
		},
	}

	err := AddSource(SourceAddRequest{
		NamespaceName: "fresh-ns",
		SourceURL:     "https://github.com/fresh/repo",
		SourceType:    "git",
		Manifest:      manifest,
	})
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	cache, err := ReadSourcesIndex()
	if err != nil {
		t.Fatalf("ReadSourcesIndex failed: %v", err)
	}
	if cache == nil {
		t.Fatalf("expected cache to be initialized and non-nil")
	}
	if _, ok := cache.Namespaces["fresh-ns"]; !ok {
		t.Fatalf("expected fresh-ns to exist in initialized cache")
	}
}
