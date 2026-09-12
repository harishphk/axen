package core

import (
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
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

func TestPruneSkills_PreservesNonInstalledStatus(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	config := &models.Config{Targets: map[string]string{
		"test-target": targetDest,
	}}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	statuses := []string{"conflict", "skipped", "untracked_conflict"}
	for _, st := range statuses {
		skillName := "skill-" + st
		skillDir := filepath.Join(targetDest, skillName)
		_ = os.MkdirAll(skillDir, 0755)
		_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+st), 0644)

		oldState := map[string]models.LockfileSkill{
			skillName: {Targets: []string{"test-target"}},
		}
		newResults := []InstallResult{
			{
				SkillName: skillName,
				Status:    st,
			},
		}

		prunes, err := PruneSkills(oldState, newResults, []string{"test-target"}, false, nil)
		if err != nil {
			t.Fatalf("[%s] PruneSkills failed: %v", st, err)
		}
		if len(prunes) != 0 {
			t.Fatalf("[%s] Expected 0 prunes, got %v", st, prunes)
		}

		// Ensure skill directory still exists
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); os.IsNotExist(err) {
			t.Fatalf("[%s] Expected skill file to still exist on disk, but it was deleted", st)
		}
	}
}

func TestPruneSkills_PreservesSkippedDestinations(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	targetDest1 := filepath.Join(tempHome, "target1")
	targetDest2 := filepath.Join(tempHome, "target2")
	config := &models.Config{Targets: map[string]string{
		"target1": targetDest1,
		"target2": targetDest2,
	}}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	skillDir1 := filepath.Join(targetDest1, "my-skill")
	skillDir2 := filepath.Join(targetDest2, "my-skill")
	_ = os.MkdirAll(skillDir1, 0755)
	_ = os.MkdirAll(skillDir2, 0755)
	_ = os.WriteFile(filepath.Join(skillDir1, "SKILL.md"), []byte("# target 1"), 0644)
	_ = os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte("# target 2"), 0644)

	oldState := map[string]models.LockfileSkill{
		"my-skill": {Targets: []string{"target1", "target2"}},
	}
	// Status is installed because target1 succeeded, but target2 was skipped due to conflict
	newResults := []InstallResult{
		{
			SkillName: "my-skill",
			Status:    "installed",
			Destinations: []Destination{
				{Target: "target1", Path: skillDir1},
			},
			SkippedDestinations: []Destination{
				{Target: "target2", Path: skillDir2},
			},
		},
	}

	prunes, err := PruneSkills(oldState, newResults, []string{"target1", "target2"}, false, nil)
	if err != nil {
		t.Fatalf("PruneSkills failed: %v", err)
	}
	if len(prunes) != 0 {
		t.Fatalf("Expected 0 prunes, got %v", prunes)
	}
	if _, err := os.Stat(filepath.Join(skillDir2, "SKILL.md")); os.IsNotExist(err) {
		t.Fatalf("target2 skill was deleted from disk!")
	}
}
