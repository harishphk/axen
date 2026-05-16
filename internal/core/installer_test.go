package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/sources"
	"context"
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

func TestInstallSkillsJailCheck(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	_ = os.MkdirAll(targetDest, 0755)

	config := &models.Config{
		Targets: map[string]string{
			"test-target": targetDest,
		},
	}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	sourceDir := filepath.Join(tempHome, "source")
	_ = os.MkdirAll(filepath.Join(sourceDir, "skill1"), 0755)
	_ = os.WriteFile(filepath.Join(sourceDir, "skill1", "SKILL.md"), []byte("skill 1"), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["skill1"] = models.SkillEntry{
		Path:         "skill1",
		PathOverride: "../../outside-path", // escapes!
	}

	lockfile := models.NewLockfile()
	options := InstallOptions{
		Targets: []string{"test-target"},
	}

	results, err := InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("InstallSkills failed: %v", err)
	}

	// Verify that the skill is installed, but it has 0 destinations because it was skipped
	for _, res := range results {
		if res.SkillName == "skill1" {
			if len(res.Destinations) != 0 {
				t.Fatalf("Expected 0 destinations for jailed path, got %d", len(res.Destinations))
			}
		}
	}
}

func TestInstallSkillsRollback(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	_ = os.MkdirAll(targetDest, 0755)

	// Create a file at test-target-dest2, which will cause mkdir/copy to fail
	targetDest2File := filepath.Join(tempHome, "test-target-dest2")
	_ = os.WriteFile(targetDest2File, []byte(""), 0644)

	config := &models.Config{
		Targets: map[string]string{
			"test-target":  targetDest,
			"test-target2": targetDest2File,
		},
	}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	sourceDir := filepath.Join(tempHome, "source")
	_ = os.MkdirAll(filepath.Join(sourceDir, "skill-ok"), 0755)
	_ = os.WriteFile(filepath.Join(sourceDir, "skill-ok", "SKILL.md"), []byte("ok"), 0644)

	_ = os.MkdirAll(filepath.Join(sourceDir, "skill-bad"), 0755)
	_ = os.WriteFile(filepath.Join(sourceDir, "skill-bad", "SKILL.md"), []byte("bad"), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["skill-ok"] = models.SkillEntry{
		Path:    "skill-ok",
		Targets: []string{"test-target"},
	}
	manifest.Skills["skill-bad"] = models.SkillEntry{
		Path:    "skill-bad",
		Targets: []string{"test-target2"},
	}

	lockfile := models.NewLockfile()
	options := InstallOptions{}

	_, err := InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err == nil {
		t.Fatal("Expected InstallSkills to fail, but it succeeded")
	}

	// Verify that skill-ok was rolled back and does not exist in test-target
	okPath := filepath.Join(targetDest, "skill-ok")
	if _, err := os.Stat(okPath); !os.IsNotExist(err) {
		t.Errorf("Expected skill-ok to be rolled back, but it exists at %s", okPath)
	}
}

func TestReconcile_TargetScopeOverride(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()

	targetDest := filepath.Join(tempHome, "test-target-dest")
	targetDest2 := filepath.Join(tempHome, "test-target-dest2")
	_ = os.MkdirAll(targetDest, 0755)
	_ = os.MkdirAll(targetDest2, 0755)

	config := &models.Config{
		Targets: map[string]string{
			"test-target":  targetDest,
			"test-target2": targetDest2,
		},
	}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	sourceDir := filepath.Join(tempHome, "source")
	_ = os.MkdirAll(filepath.Join(sourceDir, "skill1"), 0755)
	_ = os.WriteFile(filepath.Join(sourceDir, "skill1", "SKILL.md"), []byte("skill 1"), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["skill1"] = models.SkillEntry{
		Path: "skill1",
	}

	lockfile := models.NewLockfile()

	// 1. Initial install to set namespace defaults
	intent := Intent{
		ExplicitSkills: []string{"skill1"},
		Targets:        []string{"test-target", "test-target2"},
	}
	fetchResult := &sources.FetchResult{
		LocalPath: sourceDir,
		Type:      sources.SourceTypeLocal,
	}

	_, err := Reconcile(context.Background(), lockfile, "ns", "local-source", fetchResult, manifest, intent, ReconcileOpts{})
	if err != nil {
		t.Fatalf("Initial Reconcile failed: %v", err)
	}

	// Verify namespace defaults in lockfile
	nsEntry := lockfile.Namespaces["ns"]
	if len(nsEntry.Targets) != 2 || nsEntry.Targets[0] != "test-target" || nsEntry.Targets[1] != "test-target2" {
		t.Fatalf("Expected namespace targets to be [test-target, test-target2], got %v", nsEntry.Targets)
	}

	// 2. Install with TargetScope override (only test-target)
	opts := ReconcileOpts{
		TargetScope: []string{"test-target"},
		UpdateMode:  true, // re-process to apply target scope
	}
	// Keep the same intent (namespace defaults are still [test-target, test-target2])
	_, err = Reconcile(context.Background(), lockfile, "ns", "local-source", fetchResult, manifest, intent, opts)
	if err != nil {
		t.Fatalf("Reconcile with TargetScope failed: %v", err)
	}

	// Verify namespace defaults are UNCHANGED in lockfile
	nsEntry = lockfile.Namespaces["ns"]
	if len(nsEntry.Targets) != 2 || nsEntry.Targets[0] != "test-target" || nsEntry.Targets[1] != "test-target2" {
		t.Fatalf("Namespace targets were overwritten! Expected [test-target, test-target2], got %v", nsEntry.Targets)
	}

	// Verify that skill1 has a target override of [test-target]
	skill1Info := nsEntry.Skills.Installed["skill1"]
	if len(skill1Info.Targets) != 1 || skill1Info.Targets[0] != "test-target" {
		t.Fatalf("Expected skill1 to have override targets [test-target], got %v", skill1Info.Targets)
	}
}

func TestUntrackedConflicts(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	resolvers.ResetTargetPathCache()

	targetDest := filepath.Join(tempHome, "cursor-dir")
	targetDest2 := filepath.Join(tempHome, "claude-dir")

	config := &models.Config{
		Targets: map[string]string{
			"cursor": targetDest,
			"claude": targetDest2,
		},
	}
	_ = WriteConfig(config)
	resolvers.ResetTargetPathCache()

	sourceDir := filepath.Join(tempHome, "source")
	skillDir := filepath.Join(sourceDir, "skill1")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("my skill content"), 0644)

	manifest := models.NewManifest("ns")
	manifest.Skills["skill1"] = models.SkillEntry{
		Path: "skill1",
	}

	// Case 1: Untracked directory already exists on disk.
	untrackedDir := filepath.Join(targetDest, "skill1")
	_ = os.MkdirAll(untrackedDir, 0755)
	_ = os.WriteFile(filepath.Join(untrackedDir, "manual.txt"), []byte("manual content"), 0644)

	lockfile := models.NewLockfile()

	// 1a. Install without Force and without Resolver (non-interactive).
	options := InstallOptions{
		Force:   false,
		Targets: []string{"cursor", "claude"},
	}

	results, err := InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var res1 InstallResult
	found := false
	for _, r := range results {
		if r.SkillName == "skill1" {
			res1 = r
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected results for skill1")
	}

	if len(res1.Destinations) != 1 || res1.Destinations[0].Target != "claude" {
		t.Errorf("Expected Destinations to only have claude, got %v", res1.Destinations)
	}
	if len(res1.SkippedDestinations) != 1 || res1.SkippedDestinations[0].Target != "cursor" {
		t.Errorf("Expected SkippedDestinations to have cursor, got %v", res1.SkippedDestinations)
	}

	if _, err := os.Stat(filepath.Join(untrackedDir, "manual.txt")); os.IsNotExist(err) {
		t.Errorf("Expected manual.txt to exist in cursor target")
	}

	// 1b. Install with Resolver deciding to Overwrite.
	options.UntrackedConflictResolver = func(conflicts []UntrackedConflict) (map[string]bool, error) {
		decisions := make(map[string]bool)
		for _, c := range conflicts {
			decisions[c.SkillName+":"+c.Target] = true
		}
		return decisions, nil
	}

	results, err = InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	found = false
	for _, r := range results {
		if r.SkillName == "skill1" {
			res1 = r
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected results for skill1")
	}

	if len(res1.Destinations) != 2 {
		t.Errorf("Expected both destinations to be installed, got %v", res1.Destinations)
	}
	if len(res1.SkippedDestinations) != 0 {
		t.Errorf("Expected no skipped destinations, got %v", res1.SkippedDestinations)
	}

	if _, err := os.Stat(filepath.Join(untrackedDir, "manual.txt")); !os.IsNotExist(err) {
		t.Errorf("Expected manual.txt to be overwritten")
	}

	// 1c. Install with Force: true.
	_ = os.WriteFile(filepath.Join(untrackedDir, "manual.txt"), []byte("manual content"), 0644)
	options.Force = true
	options.UntrackedConflictResolver = nil

	results, err = InstallSkills(sourceDir, manifest, lockfile, "ns", options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	found = false
	for _, r := range results {
		if r.SkillName == "skill1" {
			res1 = r
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected results for skill1")
	}

	if len(res1.Destinations) != 2 {
		t.Errorf("Expected both destinations to be installed with force, got %v", res1.Destinations)
	}

	if _, err := os.Stat(filepath.Join(untrackedDir, "manual.txt")); !os.IsNotExist(err) {
		t.Errorf("Expected manual.txt to be overwritten under force")
	}
}

