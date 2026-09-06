package services

import (
	"os"
	"testing"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
)

func setupTestEnvironment(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	t.Cleanup(func() { _ = os.Unsetenv("AXEN_TEST_HOME") })

	// Init dummy lockfile
	lockfile := models.NewLockfile()
	_ = core.WriteLockfile(lockfile)
	
	return tmpDir
}

func TestSourceService_Add(t *testing.T) {
	setupTestEnvironment(t)
	svc := &SourceService{}

	err := svc.Add(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     "https://github.com/test/repo",
		SourceType:    "git",
		UpdatePolicy:  "daily",
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	lockfile, _ := core.ReadLockfile()
	if _, ok := lockfile.Namespaces["test-ns"]; !ok {
		t.Errorf("expected namespace to be added")
	}
}

func TestSourceService_SetPolicy(t *testing.T) {
	setupTestEnvironment(t)
	svc := &SourceService{}

	_ = svc.Add(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     "https://github.com/test/repo",
		SourceType:    "git",
	})
	
	err := svc.SetPolicy("test-ns", "daily")
	if err != nil {
		t.Fatalf("SetPolicy failed: %v", err)
	}

	lockfile, _ := core.ReadLockfile()
	if lockfile.Namespaces["test-ns"].UpdatePolicy != "daily" {
		t.Errorf("expected update policy to be daily")
	}

	// Test missing namespace
	err = svc.SetPolicy("non-existent", "daily")
	if err == nil {
		t.Errorf("expected error for non-existent namespace")
	}
}
