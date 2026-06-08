package resolvers

import (
	"github.com/harishphk/axen/internal/models"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTargets(t *testing.T) {
	// Create a temp home dir
	tmpDir, err := os.MkdirTemp("", "axen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	axenDir := filepath.Join(tmpDir, ".axen")
	_ = os.MkdirAll(axenDir, 0755)

	configPath := filepath.Join(axenDir, "axen-config.json")

	// Create a mock axen-config.json
	mockConfig := models.Config{
		Targets: map[string]string{
			"custom": "~/custom/skills/",
			"gemini": "~/custom/gemini/skills/",
		},
	}

	configData, _ := json.Marshal(mockConfig)
	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Reset cache to ensure we read from config
	ResetTargetPathCache()

	paths := GetTargetPaths()

	// Check custom target
	if paths["custom"] != "~/custom/skills/" {
		t.Errorf("Expected custom target path to be ~/custom/skills/, got %s", paths["custom"])
	}

	// Check overwritten default target
	if paths["gemini"] != "~/custom/gemini/skills/" {
		t.Errorf("Expected gemini target path to be ~/custom/gemini/skills/, got %s", paths["gemini"])
	}

	// Test ResolveTargetPath
	resolved := ResolveTargetPath("custom")
	if resolved == nil {
		t.Fatal("Expected resolved to not be nil")
	}
	expectedResolved := filepath.Join(tmpDir, "custom/skills/")
	if *resolved != expectedResolved {
		t.Errorf("Expected resolved path to be %s, got %s", expectedResolved, *resolved)
	}

	resolvedNil := ResolveTargetPath("nonexistent")
	if resolvedNil != nil {
		t.Errorf("Expected nil for nonexistent target")
	}

	// Test IsKnownTarget
	if !IsKnownTarget("custom") {
		t.Errorf("Expected custom to be a known target")
	}
	if IsKnownTarget("nonexistent") {
		t.Errorf("Expected nonexistent to NOT be a known target")
	}

	// Test GetKnownTargets
	known := GetKnownTargets()
	found := false
	for _, k := range known {
		if k == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected custom to be in known targets")
	}

	// Test GetTargetPaths caching
	cachedPaths := GetTargetPaths()
	if len(cachedPaths) != len(paths) {
		t.Errorf("Expected cached paths to match")
	}

	// Test broken config scenario (unmarshal error)
	ResetTargetPathCache()
	_ = os.WriteFile(configPath, []byte("{invalid-json}"), 0644)

	fallbackPaths := GetTargetPaths()
	if fallbackPaths["gemini"] == "~/custom/gemini/skills/" {
		t.Errorf("Should have fallen back to default targets when config is broken")
	}
}
