package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"os"
	"path/filepath"
	"testing"
)

func TestInstaller(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()

	axenDir := resolvers.GetAxenDir()
	_ = os.MkdirAll(axenDir, 0755)

	targetDest := filepath.Join(tempHome, "test-target-dest")
	targetDest2 := filepath.Join(tempHome, "test-target-dest2")

	// Create config
	config := &models.Config{
		Targets: map[string]string{
			"test-target":  targetDest,
			"test-target2": targetDest2,
		},
	}
	err := WriteConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	resolvers.ResetTargetPathCache()

	// Mock source dir with fake skill
	sourceDir := filepath.Join(tempHome, "source")
	skillDir := filepath.Join(sourceDir, "skill1")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill 1"), 0644)

	skill2Dir := filepath.Join(sourceDir, "skill2")
	_ = os.MkdirAll(skill2Dir, 0755)
	_ = os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("skill 2"), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["skill1"] = models.SkillEntry{
		Path: "skill1",
	}
	manifest.Skills["skill2"] = models.SkillEntry{
		Path:    "skill2",
		Targets: []string{"test-target2"},
	}
	manifest.Skills["missing-skill"] = models.SkillEntry{
		Path: "missing",
	}

	lockfile := models.NewLockfile()
	lockfile = UpsertNamespace(lockfile, "other-ns", models.NamespaceEntry{
		Skills: models.NamespaceSkills{
			Installed: map[string]models.LockfileSkill{
				"skill1": {},
			},
		},
	})

	options := InstallOptions{
		Force:            false,
		DryRun:           false,
		Targets:          []string{"test-target", "test-target2"},
		ConflictStrategy: "keep",
	}

	// 1. InstallSkills: skill1 conflict, skill2 installs, missing skips
	results, err := InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	foundSkipped := false
	foundConflict := false
	foundInstalled := false
	for _, res := range results {
		if res.SkillName == "missing-skill" && res.Status == "skipped" {
			foundSkipped = true
		}
		if res.SkillName == "skill1" && res.Status == "conflict" {
			foundConflict = true
		}
		if res.SkillName == "skill2" && res.Status == "installed" {
			foundInstalled = true
			if len(res.Destinations) != 1 || res.Destinations[0].Target != "test-target2" {
				t.Fatalf("Expected skill2 in test-target2, got %v", res.Destinations)
			}
		}
	}
	if !foundSkipped || !foundConflict || !foundInstalled {
		t.Fatalf("Expected skipped, conflict, and installed, got %v", results)
	}

	// 2. InstallSkills with Force
	options.Force = true
	_, err = InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 3. InstallSkills with Unknown target
	options.Targets = []string{"unknown"}
	_, err = InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err == nil {
		t.Fatalf("Expected error for unknown target")
	}

	// 4. UninstallSkillFromTargets
	removed, err := UninstallSkillFromTargets("skill1", []string{"test-target"}, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(removed) != 1 || removed[0].Target != "test-target" {
		t.Fatalf("Expected skill1 removed from test-target, got %v", removed)
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill1")); !os.IsNotExist(err) {
		t.Fatalf("Expected skill1 dir to be removed")
	}

	// Test uninstall invalid name
	_, err = UninstallSkillFromTargets("../skill1", []string{"test-target"}, false)
	if err == nil {
		t.Fatalf("Expected error for invalid skill name")
	}

	// 5. PruneSkills
	oldState := map[string]models.LockfileSkill{
		"skill-to-remove": {Targets: []string{"test-target"}},
		"skill-to-keep":   {Targets: []string{"test-target"}},
	}
	// Create mock skill dirs to prune
	_ = os.MkdirAll(filepath.Join(targetDest, "skill-to-remove"), 0755)

	newResults := []InstallResult{
		{
			SkillName: "skill-to-keep",
			Status:    "installed",
			Destinations: []Destination{
				{Target: "test-target", Path: filepath.Join(targetDest, "skill-to-keep")},
			},
		},
	}

	prunes, err := PruneSkills(oldState, newResults, []string{"test-target"}, false, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(prunes) != 1 || prunes[0].SkillName != "skill-to-remove" {
		t.Fatalf("Expected skill-to-remove pruned, got %v", prunes)
	}
}

func TestInstallDryRun(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()
	config := &models.Config{Targets: map[string]string{"t": t.TempDir()}}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	src := t.TempDir()
	_ = os.MkdirAll(filepath.Join(src, "s1"), 0755)
	_ = os.WriteFile(filepath.Join(src, "s1", "f"), []byte(""), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["s1"] = models.SkillEntry{Path: "s1"}
	lockfile := models.NewLockfile()

	results, err := InstallSkills(src, manifest, lockfile, "ns", InstallOptions{DryRun: true, Targets: []string{"t"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "installed" {
		t.Fatal("Expected installed")
	}

	// Uninstall dry run
	removed, err := UninstallSkillFromTargets("s1", []string{"t"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 0 {
		t.Fatalf("Dry run shouldn't actually find the skill path (not created), got %v", removed)
	}
}
