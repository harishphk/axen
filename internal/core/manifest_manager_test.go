package core

import (
	"github.com/harishphk/axen/internal/models"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestManager(t *testing.T) {
	tempDir := t.TempDir()

	// 1. HasManifest / ReadManifest when missing
	if HasManifest(tempDir) {
		t.Fatalf("Expected false")
	}
	_, err := ReadManifest(tempDir)
	if err == nil {
		t.Fatalf("Expected error when missing")
	}

	// 2. GenerateManifest
	scanned := []ScannedSkill{
		{
			RelativePath: "skill1",
			Frontmatter: models.Frontmatter{
				Name:     "s1",
				Targets:  []string{"t1"},
				Metadata: map[string]string{"version": "1.0"},
			},
		},
	}
	manifest := GenerateManifest("test-ns", scanned, []string{"default"})
	if manifest.Name != "test-ns" {
		t.Fatalf("Expected test-ns")
	}
	if len(manifest.Targets) != 1 || manifest.Targets[0] != "default" {
		t.Fatalf("Expected [default]")
	}
	if manifest.Skills["s1"].Version != "1.0" || manifest.Skills["s1"].Path != "skill1" || len(manifest.Skills["s1"].Targets) != 1 || manifest.Skills["s1"].Targets[0] != "t1" {
		t.Fatalf("Unexpected generated skill entry: %v", manifest.Skills["s1"])
	}

	// 3. WriteManifest
	err = WriteManifest(tempDir, manifest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !HasManifest(tempDir) {
		t.Fatalf("Expected true")
	}

	// 4. ReadManifest when present
	readMan, err := ReadManifest(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if readMan.Name != "test-ns" {
		t.Fatalf("Expected test-ns")
	}

	// 5. Corrupted manifest
	err = os.WriteFile(filepath.Join(tempDir, ManifestFilename), []byte("{invalid"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReadManifest(tempDir)
	if err == nil {
		t.Fatalf("Expected error on invalid json")
	}

	// 6. MergeManifest
	existing := models.NewManifest("test-ns")
	existing.Targets = []string{"t1"}
	existing.Skills["old-skill"] = models.SkillEntry{
		Path:    "old",
		Version: "0.9",
	}

	newScanned := []ScannedSkill{
		{
			RelativePath: "old-updated",
			Frontmatter: models.Frontmatter{
				Name: "old-skill", // updating old
			},
		},
		{
			RelativePath: "new",
			Frontmatter: models.Frontmatter{
				Name:     "new-skill",
				Metadata: map[string]string{"version": "2.0"},
			},
		},
	}

	merged := MergeManifest(existing, newScanned)
	if merged.Name != "test-ns" {
		t.Fatalf("Expected test-ns")
	}
	if len(merged.Targets) != 1 || merged.Targets[0] != "t1" {
		t.Fatalf("Expected [t1]")
	}
	if len(merged.Skills) != 2 {
		t.Fatalf("Expected 2 skills, got %d", len(merged.Skills))
	}
	if merged.Skills["old-skill"].Path != "old-updated" || merged.Skills["old-skill"].Version != "0.9" {
		t.Fatalf("Expected path updated but version kept. Got: %v", merged.Skills["old-skill"])
	}
	if merged.Skills["new-skill"].Version != "2.0" {
		t.Fatalf("Expected new skill version 2.0")
	}
}
