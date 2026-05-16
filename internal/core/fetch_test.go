package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchAndResolveLocal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "axen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	_ = os.Setenv("AXEN_TEST_HOME", tempDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	// Create a local fake skill repo
	skillDir := filepath.Join(tempDir, "my-local-source")
	err = os.MkdirAll(filepath.Join(skillDir, "my-skill"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(skillDir, "my-skill", "SKILL.md"), []byte("---\nname: my-skill\ndescription: A test skill\n---\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	res, manifest, err := FetchAndResolve(context.Background(), skillDir, "local-ns")
	if err != nil {
		t.Fatalf("FetchAndResolve failed: %v", err)
	}

	if res == nil {
		t.Fatal("Expected non-nil FetchResult")
	}
	if res.LocalPath != skillDir {
		t.Errorf("Expected LocalPath %s, got %s", skillDir, res.LocalPath)
	}

	if manifest == nil {
		t.Fatal("Expected non-nil manifest")
	}
	if _, ok := manifest.Skills["my-skill"]; !ok {
		t.Errorf("Expected my-skill in manifest")
	}
}
