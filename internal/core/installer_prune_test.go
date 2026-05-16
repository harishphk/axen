package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallerPruneMore(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	config := &models.Config{Targets: map[string]string{"test-target": targetDest}}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	// Create a mock skill directory
	skillDir := filepath.Join(targetDest, "filtered-out")
	_ = os.MkdirAll(skillDir, 0755)

	oldState := map[string]models.LockfileSkill{
		"filtered-out":  {Targets: []string{"test-target"}},
		"removed-skill": {Targets: []string{"test-target"}},
	}
	_ = os.MkdirAll(filepath.Join(targetDest, "removed-skill"), 0755)

	// Test PruneSkills with filter
	prunes, err := PruneSkills(oldState, []InstallResult{}, []string{"test-target"}, false, []string{"removed-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if len(prunes) != 1 || prunes[0].SkillName != "removed-skill" {
		t.Fatalf("Expected only removed-skill to be pruned, got %v", prunes)
	}
}

func TestPruneExisting(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	targetDest2 := filepath.Join(tempHome, "test-target-dest2")
	config := &models.Config{Targets: map[string]string{
		"test-target":  targetDest,
		"test-target2": targetDest2,
	}}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	// Skill exists in old state but now only installs to one target
	oldState := map[string]models.LockfileSkill{
		"skill-partial": {Targets: []string{"test-target", "test-target2"}},
	}
	_ = os.MkdirAll(filepath.Join(targetDest, "skill-partial"), 0755)
	_ = os.MkdirAll(filepath.Join(targetDest2, "skill-partial"), 0755)

	newResults := []InstallResult{
		{
			SkillName: "skill-partial",
			Status:    "installed",
			Destinations: []Destination{
				{Target: "test-target", Path: filepath.Join(targetDest, "skill-partial")},
			},
		},
	}

	prunes, err := PruneSkills(oldState, newResults, []string{"test-target", "test-target2"}, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(prunes) != 1 {
		t.Fatalf("Expected 1 prune, got %v", prunes)
	}
	if len(prunes[0].Removed) != 1 || prunes[0].Removed[0].Target != "test-target2" {
		t.Fatalf("Expected test-target2 to be removed, got %v", prunes[0].Removed)
	}
}
