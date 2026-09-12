package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
)

// ---------------------------------------------------------------------------
// TestGetIntent
// ---------------------------------------------------------------------------

func TestGetIntent(t *testing.T) {
	t.Run("namespace exists", func(t *testing.T) {
		lockfile := models.NewLockfile()
		lockfile.Namespaces["ns"] = models.NamespaceEntry{
			Bundles:        []string{"frontend"},
			ExplicitSkills: []string{"extra"},
			Excluded:       []string{"unwanted"},
			Targets:        []string{"claude"},
			SyncAll:        false,
		}

		intent := GetIntent(lockfile, "ns")
		assertEqual(t, intent.Bundles, []string{"frontend"})
		assertEqual(t, intent.ExplicitSkills, []string{"extra"})
		assertEqual(t, intent.Excluded, []string{"unwanted"})
		assertEqual(t, intent.Targets, []string{"claude"})
		if intent.SyncAll {
			t.Fatal("Expected SyncAll false")
		}
	})

	t.Run("namespace does not exist", func(t *testing.T) {
		lockfile := models.NewLockfile()
		intent := GetIntent(lockfile, "missing")
		if len(intent.Bundles) != 0 || len(intent.ExplicitSkills) != 0 {
			t.Fatal("Expected empty intent")
		}
	})
}

// ---------------------------------------------------------------------------
// TestMergeIntent
// ---------------------------------------------------------------------------

func TestMergeIntent(t *testing.T) {
	tests := []struct {
		name     string
		current  Intent
		action   IntentAction
		expected Intent
	}{
		{
			name:    "add bundles to empty intent",
			current: Intent{},
			action:  IntentAction{AddBundles: []string{"frontend", "backend"}},
			expected: Intent{
				Bundles: []string{"backend", "frontend"}, // sorted
			},
		},
		{
			name:    "add bundles with deduplication",
			current: Intent{Bundles: []string{"frontend"}},
			action:  IntentAction{AddBundles: []string{"frontend", "backend"}},
			expected: Intent{
				Bundles: []string{"backend", "frontend"},
			},
		},
		{
			name:    "remove bundles",
			current: Intent{Bundles: []string{"backend", "frontend"}},
			action:  IntentAction{RemoveBundles: []string{"frontend"}},
			expected: Intent{
				Bundles: []string{"backend"},
			},
		},
		{
			name:    "remove nonexistent bundle is no-op",
			current: Intent{Bundles: []string{"frontend"}},
			action:  IntentAction{RemoveBundles: []string{"does-not-exist"}},
			expected: Intent{
				Bundles: []string{"frontend"},
			},
		},
		{
			name:    "add explicit skills",
			current: Intent{},
			action:  IntentAction{AddSkills: []string{"skill-b", "skill-a"}},
			expected: Intent{
				ExplicitSkills: []string{"skill-a", "skill-b"}, // sorted
			},
		},
		{
			name:    "add skills with deduplication",
			current: Intent{ExplicitSkills: []string{"skill-a"}},
			action:  IntentAction{AddSkills: []string{"skill-a", "skill-b"}},
			expected: Intent{
				ExplicitSkills: []string{"skill-a", "skill-b"},
			},
		},
		{
			name:    "remove explicit skills",
			current: Intent{ExplicitSkills: []string{"skill-a", "skill-b"}},
			action:  IntentAction{RemoveSkills: []string{"skill-a"}},
			expected: Intent{
				ExplicitSkills: []string{"skill-b"},
			},
		},
		{
			name:    "adding skill removes it from excluded",
			current: Intent{Excluded: []string{"skill-a", "skill-b"}},
			action:  IntentAction{AddSkills: []string{"skill-a"}},
			expected: Intent{
				ExplicitSkills: []string{"skill-a"},
				Excluded:       []string{"skill-b"},
			},
		},
		{
			name:    "add to excluded list",
			current: Intent{},
			action:  IntentAction{AddExcluded: []string{"unwanted"}},
			expected: Intent{
				Excluded: []string{"unwanted"},
			},
		},
		{
			name:    "add to excluded with deduplication",
			current: Intent{Excluded: []string{"unwanted"}},
			action:  IntentAction{AddExcluded: []string{"unwanted", "other"}},
			expected: Intent{
				Excluded: []string{"other", "unwanted"}, // sorted
			},
		},
		{
			name:    "set targets",
			current: Intent{Targets: []string{"claude"}},
			action:  IntentAction{SetTargets: []string{"cursor", "warp"}},
			expected: Intent{
				Targets: []string{"cursor", "warp"},
			},
		},
		{
			name:    "nil SetTargets keeps existing",
			current: Intent{Targets: []string{"claude"}},
			action:  IntentAction{SetTargets: nil},
			expected: Intent{
				Targets: []string{"claude"},
			},
		},
		{
			name:    "set sync all true",
			current: Intent{SyncAll: false},
			action:  IntentAction{SetSyncAll: boolPtr(true)},
			expected: Intent{
				SyncAll: true,
			},
		},
		{
			name:    "set sync all false",
			current: Intent{SyncAll: true},
			action:  IntentAction{SetSyncAll: boolPtr(false)},
			expected: Intent{
				SyncAll: false,
			},
		},
		{
			name:    "nil SetSyncAll keeps existing",
			current: Intent{SyncAll: true},
			action:  IntentAction{},
			expected: Intent{
				SyncAll: true,
			},
		},
		{
			name: "complex: add bundle + remove skill + set targets",
			current: Intent{
				Bundles:        []string{"frontend"},
				ExplicitSkills: []string{"skill-a", "skill-b"},
				Targets:        []string{"claude"},
			},
			action: IntentAction{
				AddBundles:   []string{"backend"},
				RemoveSkills: []string{"skill-a"},
				SetTargets:   []string{"cursor"},
			},
			expected: Intent{
				Bundles:        []string{"backend", "frontend"},
				ExplicitSkills: []string{"skill-b"},
				Targets:        []string{"cursor"},
			},
		},
		{
			name:    "does not mutate original intent",
			current: Intent{Bundles: []string{"frontend"}, ExplicitSkills: []string{"skill-a"}},
			action:  IntentAction{AddBundles: []string{"backend"}, RemoveSkills: []string{"skill-a"}},
			expected: Intent{
				Bundles: []string{"backend", "frontend"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original for mutation check
			origBundles := copyStrings(tt.current.Bundles)

			result := MergeIntent(tt.current, tt.action)

			// Normalize nils to empty for comparison
			assertEqualNorm(t, "Bundles", result.Bundles, tt.expected.Bundles)
			assertEqualNorm(t, "ExplicitSkills", result.ExplicitSkills, tt.expected.ExplicitSkills)
			assertEqualNorm(t, "Excluded", result.Excluded, tt.expected.Excluded)
			assertEqualNorm(t, "Targets", result.Targets, tt.expected.Targets)
			if result.SyncAll != tt.expected.SyncAll {
				t.Errorf("SyncAll: got %v, want %v", result.SyncAll, tt.expected.SyncAll)
			}

			// Verify original was not mutated
			if tt.name == "does not mutate original intent" {
				assertEqual(t, tt.current.Bundles, origBundles)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveIntent
// ---------------------------------------------------------------------------

func TestResolveIntent(t *testing.T) {
	manifest := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"skill-a": {Path: "."},
			"skill-b": {Path: "."},
			"skill-c": {Path: "."},
			"skill-d": {Path: "."},
		},
		Bundles: map[string]models.Bundle{
			"frontend": {Skills: []string{"skill-a", "skill-b"}},
			"backend":  {Skills: []string{"skill-c"}},
		},
	}

	tests := []struct {
		name     string
		intent   Intent
		expected map[string]bool
	}{
		{
			name:     "single bundle",
			intent:   Intent{Bundles: []string{"frontend"}},
			expected: map[string]bool{"skill-a": true, "skill-b": true},
		},
		{
			name:     "multiple bundles",
			intent:   Intent{Bundles: []string{"frontend", "backend"}},
			expected: map[string]bool{"skill-a": true, "skill-b": true, "skill-c": true},
		},
		{
			name:     "explicit skills only",
			intent:   Intent{ExplicitSkills: []string{"skill-d"}},
			expected: map[string]bool{"skill-d": true},
		},
		{
			name:     "bundle + explicit overlap",
			intent:   Intent{Bundles: []string{"frontend"}, ExplicitSkills: []string{"skill-a", "skill-d"}},
			expected: map[string]bool{"skill-a": true, "skill-b": true, "skill-d": true},
		},
		{
			name:     "sync all",
			intent:   Intent{SyncAll: true},
			expected: map[string]bool{"skill-a": true, "skill-b": true, "skill-c": true, "skill-d": true},
		},
		{
			name:     "sync all with excluded",
			intent:   Intent{SyncAll: true, Excluded: []string{"skill-a", "skill-c"}},
			expected: map[string]bool{"skill-b": true, "skill-d": true},
		},
		{
			name:     "bundle with excluded",
			intent:   Intent{Bundles: []string{"frontend"}, Excluded: []string{"skill-a"}},
			expected: map[string]bool{"skill-b": true},
		},
		{
			name:     "bundle not in manifest is ignored",
			intent:   Intent{Bundles: []string{"does-not-exist"}},
			expected: map[string]bool{},
		},
		{
			name:     "empty intent",
			intent:   Intent{},
			expected: map[string]bool{},
		},
		{
			name:     "explicit skill not in manifest still appears in desired",
			intent:   Intent{ExplicitSkills: []string{"not-in-manifest"}},
			expected: map[string]bool{"not-in-manifest": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := ResolveIntent(tt.intent, manifest)

			if len(resolved.DesiredSkills) != len(tt.expected) {
				t.Fatalf("DesiredSkills length: got %d, want %d\ngot: %v\nwant: %v",
					len(resolved.DesiredSkills), len(tt.expected), resolved.DesiredSkills, tt.expected)
			}
			for skill := range tt.expected {
				if !resolved.DesiredSkills[skill] {
					t.Errorf("Expected skill %q in desired state", skill)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDiffState
// ---------------------------------------------------------------------------

func TestDiffState(t *testing.T) {
	tests := []struct {
		name            string
		current         map[string]models.LockfileSkill
		desired         map[string]bool
		expectInstall   []string
		expectPrune     []string
		expectUnchanged []string
	}{
		{
			name:            "fresh install — everything is new",
			current:         map[string]models.LockfileSkill{},
			desired:         map[string]bool{"a": true, "b": true},
			expectInstall:   []string{"a", "b"},
			expectPrune:     nil,
			expectUnchanged: nil,
		},
		{
			name:            "nothing changed",
			current:         map[string]models.LockfileSkill{"a": {}, "b": {}},
			desired:         map[string]bool{"a": true, "b": true},
			expectInstall:   nil,
			expectPrune:     nil,
			expectUnchanged: []string{"a", "b"},
		},
		{
			name:            "everything removed",
			current:         map[string]models.LockfileSkill{"a": {}, "b": {}},
			desired:         map[string]bool{},
			expectInstall:   nil,
			expectPrune:     []string{"a", "b"},
			expectUnchanged: nil,
		},
		{
			name:            "mixed: some new, some existing, some removed",
			current:         map[string]models.LockfileSkill{"a": {}, "b": {}, "c": {}},
			desired:         map[string]bool{"b": true, "c": true, "d": true},
			expectInstall:   []string{"d"},
			expectPrune:     []string{"a"},
			expectUnchanged: []string{"b", "c"},
		},
		{
			name:            "both empty",
			current:         map[string]models.LockfileSkill{},
			desired:         map[string]bool{},
			expectInstall:   nil,
			expectPrune:     nil,
			expectUnchanged: nil,
		},
		{
			name:            "nil current",
			current:         nil,
			desired:         map[string]bool{"a": true},
			expectInstall:   []string{"a"},
			expectPrune:     nil,
			expectUnchanged: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := ResolvedState{DesiredSkills: tt.desired}
			plan := DiffState(tt.current, resolved)

			assertEqualSorted(t, "ToInstall", plan.ToInstall, tt.expectInstall)
			assertEqualSorted(t, "ToPrune", plan.ToPrune, tt.expectPrune)
			assertEqualSorted(t, "Unchanged", plan.Unchanged, tt.expectUnchanged)
		})
	}
}

// ---------------------------------------------------------------------------
// TestMergeIntentIdempotent
// ---------------------------------------------------------------------------

func TestMergeIntentIdempotent(t *testing.T) {
	intent := Intent{
		Bundles:        []string{"frontend"},
		ExplicitSkills: []string{"skill-a"},
		Targets:        []string{"claude"},
	}

	action := IntentAction{
		AddBundles: []string{"frontend"},
		AddSkills:  []string{"skill-a"},
	}

	result := MergeIntent(intent, action)

	assertEqual(t, result.Bundles, []string{"frontend"})
	assertEqual(t, result.ExplicitSkills, []string{"skill-a"})
	assertEqual(t, result.Targets, []string{"claude"})
}

// ---------------------------------------------------------------------------
// TestEndToEndScenarios
// ---------------------------------------------------------------------------

func TestEndToEndScenario_InstallThenRemoveBundle(t *testing.T) {
	manifest := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"skill-a": {Path: "."},
			"skill-b": {Path: "."},
			"skill-c": {Path: "."},
		},
		Bundles: map[string]models.Bundle{
			"frontend": {Skills: []string{"skill-a", "skill-b"}},
		},
	}

	// Step 1: Install bundle "frontend" + explicit "skill-c"
	intent := MergeIntent(Intent{}, IntentAction{
		AddBundles: []string{"frontend"},
		AddSkills:  []string{"skill-c"},
		SetTargets: []string{"claude"},
	})
	resolved := ResolveIntent(intent, manifest)
	plan := DiffState(nil, resolved)

	if len(plan.ToInstall) != 3 {
		t.Fatalf("Step 1: expected 3 to install, got %v", plan.ToInstall)
	}
	if len(plan.ToPrune) != 0 {
		t.Fatalf("Step 1: expected 0 to prune, got %v", plan.ToPrune)
	}

	// Simulate installed state after step 1
	installed := map[string]models.LockfileSkill{
		"skill-a": {}, "skill-b": {}, "skill-c": {},
	}

	// Step 2: Remove bundle "frontend"
	intent2 := MergeIntent(intent, IntentAction{
		RemoveBundles: []string{"frontend"},
	})
	resolved2 := ResolveIntent(intent2, manifest)
	plan2 := DiffState(installed, resolved2)

	// skill-a and skill-b should be pruned (were only in bundle)
	// skill-c should stay (explicit)
	assertEqualSorted(t, "Step2 ToPrune", plan2.ToPrune, []string{"skill-a", "skill-b"})
	assertEqualSorted(t, "Step2 Unchanged", plan2.Unchanged, []string{"skill-c"})
	if len(plan2.ToInstall) != 0 {
		t.Fatalf("Step 2: expected 0 to install, got %v", plan2.ToInstall)
	}
}

func TestEndToEndScenario_UpdateChangesBundle(t *testing.T) {
	// Old manifest: frontend = [skill-a]
	manifestV1 := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"skill-a": {Path: "."},
			"dummy":   {Path: "."},
		},
		Bundles: map[string]models.Bundle{
			"frontend": {Skills: []string{"skill-a"}},
			"default":  {Skills: []string{"dummy"}, IsDefault: true},
		},
	}

	// Install frontend + default
	intent := MergeIntent(Intent{}, IntentAction{
		AddBundles: []string{"frontend", "default"},
		SetTargets: []string{"claude"},
	})
	resolved := ResolveIntent(intent, manifestV1)
	plan := DiffState(nil, resolved)

	assertEqual(t, sortedCopy(plan.ToInstall), []string{"dummy", "skill-a"})

	// Simulate installed
	installed := map[string]models.LockfileSkill{
		"skill-a": {}, "dummy": {},
	}

	// New manifest: frontend = [skill-b], no default bundle
	manifestV2 := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"skill-b": {Path: "."},
			"skill-c": {Path: "."},
		},
		Bundles: map[string]models.Bundle{
			"frontend": {Skills: []string{"skill-b"}},
		},
	}

	// Update: same intent, new manifest
	resolved2 := ResolveIntent(intent, manifestV2)
	plan2 := DiffState(installed, resolved2)

	// skill-b is new (in frontend now)
	// skill-a is pruned (removed from frontend)
	// dummy is pruned (default bundle gone from manifest)
	assertEqualSorted(t, "Update ToInstall", plan2.ToInstall, []string{"skill-b"})
	assertEqualSorted(t, "Update ToPrune", plan2.ToPrune, []string{"dummy", "skill-a"})
}

func TestEndToEndScenario_SyncAllWithExclude(t *testing.T) {
	manifest := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"a": {Path: "."}, "b": {Path: "."}, "c": {Path: "."},
		},
	}

	// Install all
	intent := MergeIntent(Intent{}, IntentAction{
		SetSyncAll: boolPtr(true),
		SetTargets: []string{"claude"},
	})
	resolved := ResolveIntent(intent, manifest)
	if len(resolved.DesiredSkills) != 3 {
		t.Fatalf("Expected 3 desired skills, got %d", len(resolved.DesiredSkills))
	}

	installed := map[string]models.LockfileSkill{"a": {}, "b": {}, "c": {}}

	// Exclude "b"
	intent2 := MergeIntent(intent, IntentAction{AddExcluded: []string{"b"}})
	resolved2 := ResolveIntent(intent2, manifest)
	plan2 := DiffState(installed, resolved2)

	assertEqualSorted(t, "Unchanged", plan2.Unchanged, []string{"a", "c"})
	assertEqualSorted(t, "ToPrune", plan2.ToPrune, []string{"b"})
}

func TestEndToEndScenario_DisableSyncAllConvertsToExplicit(t *testing.T) {
	manifest := &models.Manifest{
		Skills: map[string]models.SkillEntry{
			"a": {Path: "."}, "b": {Path: "."}, "c": {Path: "."},
		},
	}

	// Start with SyncAll
	intent := Intent{SyncAll: true, Targets: []string{"claude"}}
	installed := map[string]models.LockfileSkill{"a": {}, "b": {}, "c": {}}

	// Disable SyncAll and convert remaining to explicit (removing "b")
	// This simulates the pre-hook in remove.go
	intent2 := MergeIntent(intent, IntentAction{
		SetSyncAll: boolPtr(false),
		AddSkills:  []string{"a", "c"}, // convert to explicit, minus "b"
	})

	resolved := ResolveIntent(intent2, manifest)
	plan := DiffState(installed, resolved)

	assertEqualSorted(t, "ToPrune", plan.ToPrune, []string{"b"})
	assertEqualSorted(t, "Unchanged", plan.Unchanged, []string{"a", "c"})
	if intent2.SyncAll {
		t.Fatal("SyncAll should be false")
	}
	assertEqualSorted(t, "ExplicitSkills", intent2.ExplicitSkills, []string{"a", "c"})
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func boolPtr(b bool) *bool {
	return &b
}

func sortedCopy(s []string) []string {
	c := make([]string, len(s))
	copy(c, s)
	sort.Strings(c)
	return c
}

func assertEqual(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(normalizeNil(got), normalizeNil(want)) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertEqualNorm(t *testing.T, field string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(normalizeNil(got), normalizeNil(want)) {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func assertEqualSorted(t *testing.T, label string, got, want []string) {
	t.Helper()
	g := normalizeNil(sortedCopy(got))
	w := normalizeNil(sortedCopy(want))
	if !reflect.DeepEqual(g, w) {
		t.Errorf("%s: got %v, want %v", label, g, w)
	}
}

func normalizeNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func TestPersistState_PreservesPolicyAndNilManifest(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AXEN_TEST_HOME", tmpDir)

	lockfile := models.NewLockfile()
	lockfile.Namespaces["my-ns"] = models.NamespaceEntry{
		Source:        "https://github.com/example/skills.git",
		Type:          "git",
		UpdatePolicy:  "weekly",
		LastCheckedAt: "2026-08-01T12:00:00Z",
		Targets:       []string{"claude"},
		Skills: models.NamespaceSkills{
			Installed: map[string]models.LockfileSkill{
				"old-skill": {Targets: []string{"claude"}},
			},
		},
	}

	resolved := ResolvedState{
		Intent: Intent{Targets: []string{"claude"}},
	}
	installResults := []InstallResult{
		{SkillName: "new-skill", Status: "installed", Destinations: []Destination{{Target: "claude"}}},
	}

	// Should not panic even if manifest is nil
	if err := persistState(lockfile, "my-ns", "https://github.com/example/skills.git", nil, nil, resolved, installResults, nil, lockfile.Namespaces["my-ns"].Skills.Installed); err != nil {
		t.Fatalf("persistState failed: %v", err)
	}

	entry := lockfile.Namespaces["my-ns"]
	if entry.UpdatePolicy != "weekly" {
		t.Errorf("expected UpdatePolicy to be preserved as 'weekly', got %q", entry.UpdatePolicy)
	}
	if entry.LastCheckedAt != "2026-08-01T12:00:00Z" {
		t.Errorf("expected LastCheckedAt to be preserved as '2026-08-01T12:00:00Z', got %q", entry.LastCheckedAt)
	}
	if _, ok := entry.Skills.Installed["new-skill"]; !ok {
		t.Errorf("expected new-skill to be in Installed map")
	}
}

// ---------------------------------------------------------------------------
// Deep Reconciliation Engine Tests
// ---------------------------------------------------------------------------

func setupEngineTestEnv(t *testing.T) (string, string) {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("AXEN_TEST_HOME", tempHome)
	resolvers.ResetTargetPathCache()

	axenDir := resolvers.GetAxenDir()
	_ = os.MkdirAll(axenDir, 0755)

	targetDest := filepath.Join(tempHome, "targets", "test-agent")
	_ = os.MkdirAll(targetDest, 0755)

	cfg := &models.Config{
		Targets: map[string]string{
			"test-agent": targetDest,
		},
	}
	if err := WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}
	resolvers.ResetTargetPathCache()

	lockfile := models.NewLockfile()
	if err := WriteLockfile(lockfile); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	return tempHome, targetDest
}

func createLocalTestSkill(t *testing.T, baseDir, skillName, content string) string {
	t.Helper()
	sDir := filepath.Join(baseDir, skillName)
	if err := os.MkdirAll(sDir, 0755); err != nil {
		t.Fatal(err)
	}
	body := content
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		body = fmt.Sprintf("---\nname: %s\ndescription: %s test skill\n---\n%s", skillName, skillName, content)
	}
	if err := os.WriteFile(filepath.Join(sDir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return sDir
}

func TestEngine_Inspect(t *testing.T) {
	tempHome, _ := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-inspect")
	createLocalTestSkill(t, srcDir, "skill1", "---\nname: skill1\ndescription: test\n---\n")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "ns-inspect",
		Skills: map[string]models.SkillEntry{
			"skill1": {Path: "skill1", Version: "1.0.0"},
		},
		Bundles: map[string]models.Bundle{
			"default": {IsDefault: true, Skills: []string{"skill1"}},
		},
	}
	if err := WriteManifest(srcDir, manifest); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine()

	fetchStarted := false
	fetchDone := false

	opts := InspectOptions{
		OnFetchStart: func(ns string) {
			if ns == "ns-inspect" {
				fetchStarted = true
			}
		},
		OnFetchDone: func(ns string, err error) {
			if ns == "ns-inspect" && err == nil {
				fetchDone = true
			}
		},
	}

	res, err := engine.Inspect(context.Background(), srcDir, "ns-inspect", opts)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if !fetchStarted || !fetchDone {
		t.Errorf("expected fetch progress callbacks to be called, got start=%v done=%v", fetchStarted, fetchDone)
	}
	if res == nil || res.Manifest == nil || res.FetchResult == nil {
		t.Fatalf("expected non-nil SourceManifest")
	}
	if _, ok := res.Manifest.Skills["skill1"]; !ok {
		t.Errorf("expected skill1 in manifest")
	}
	if _, ok := res.Manifest.Bundles["default"]; !ok {
		t.Errorf("expected default bundle in manifest")
	}

	// Non-existent source test
	_, err = engine.Inspect(context.Background(), filepath.Join(tempHome, "non-existent"), "missing", InspectOptions{})
	if err == nil {
		t.Errorf("expected error for non-existent source")
	}
}

func TestEngine_Install_AutoDetectDefaultBundle(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-bundle")
	createLocalTestSkill(t, srcDir, "skill1", "# skill1")
	createLocalTestSkill(t, srcDir, "skill2", "# skill2")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "src-bundle",
		Skills: map[string]models.SkillEntry{
			"skill1": {Path: "skill1"},
			"skill2": {Path: "skill2"},
		},
		Bundles: map[string]models.Bundle{
			"standard": {
				IsDefault: true,
				Skills:    []string{"skill2"},
			},
		},
	}
	_ = WriteManifest(srcDir, manifest)

	engine := NewEngine()

	defaultDetected := ""
	spec := InstallSpec{
		NamespaceName: "src-bundle",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		OnDefaultBundleDetected: func(b string) {
			defaultDetected = b
		},
	}

	res, err := engine.Install(context.Background(), spec)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if defaultDetected != "standard" {
		t.Errorf("expected default bundle 'standard' detected, got %q", defaultDetected)
	}
	if len(res.Installed) != 1 || res.Installed[0].SkillName != "skill2" {
		t.Errorf("expected only skill2 installed, got %v", res.Installed)
	}

	if _, err := os.Stat(filepath.Join(targetDest, "skill2", "SKILL.md")); err != nil {
		t.Errorf("skill2 not installed to destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill1", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("skill1 should not have been installed")
	}

	lockfile, _ := ReadLockfile()
	ns, ok := lockfile.Namespaces["src-bundle"]
	if !ok {
		t.Fatalf("expected src-bundle in lockfile")
	}
	if len(ns.Bundles) != 1 || ns.Bundles[0] != "standard" {
		t.Errorf("expected standard in lockfile bundles, got %v", ns.Bundles)
	}
}

func TestEngine_Install_ExplicitSkillsAndConflict(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir1 := filepath.Join(tempHome, "src-owner")
	createLocalTestSkill(t, srcDir1, "skill-a", "# skill-a original")

	srcDir2 := filepath.Join(tempHome, "src-conflict")
	createLocalTestSkill(t, srcDir2, "skill-a", "# skill-a conflict version")

	engine := NewEngine()

	// Initial install from ns-owner
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "ns-owner",
		SourceURL:     srcDir1,
		Targets:       []string{"test-agent"},
		AddSkills:     []string{"skill-a"},
	})
	if err != nil {
		t.Fatalf("Initial install failed: %v", err)
	}

	// Attempt install of conflicting skill-a from ns-conflict with conflict strategy keep
	res, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName:    "ns-conflict",
		SourceURL:        srcDir2,
		Targets:          []string{"test-agent"},
		AddSkills:        []string{"skill-a"},
		ConflictStrategy: "keep",
	})
	if err != nil {
		t.Fatalf("Install with keep strategy failed: %v", err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0].SkillName != "skill-a" {
		t.Errorf("expected skill-a to be in conflicts, got %v", res.Conflicts)
	}
	if len(res.Installed) != 0 {
		t.Errorf("expected 0 installed skills, got %v", res.Installed)
	}

	// Verify the original content was preserved
	content, _ := os.ReadFile(filepath.Join(targetDest, "skill-a", "SKILL.md"))
	if !strings.Contains(string(content), "original") {
		t.Errorf("expected original content to be preserved, got: %s", string(content))
	}
}

func TestEngine_Install_RollbackOnFailure(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	t.Cleanup(func() { resolvers.ResetTargetPathCache() })

	// Create a target destination that is a file, causing copy to fail
	targetDest2File := filepath.Join(tempHome, "test-target-dest2")
	_ = os.WriteFile(targetDest2File, []byte(""), 0644)

	cfg := &models.Config{
		Targets: map[string]string{
			"test-agent":   targetDest,
			"test-agent-2": targetDest2File,
		},
	}
	_ = WriteConfig(cfg)
	resolvers.ResetTargetPathCache()

	srcDir := filepath.Join(tempHome, "src-fail")
	createLocalTestSkill(t, srcDir, "skill-ok", "# skill-ok")
	createLocalTestSkill(t, srcDir, "skill-bad", "# skill-bad")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "ns-fail",
		Skills: map[string]models.SkillEntry{
			"skill-ok":  {Path: "skill-ok", Targets: []string{"test-agent"}},
			"skill-bad": {Path: "skill-bad", Targets: []string{"test-agent-2"}},
		},
	}
	_ = WriteManifest(srcDir, manifest)

	engine := NewEngine()
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "ns-fail",
		SourceURL:     srcDir,
		AddSkills:     []string{"skill-ok", "skill-bad"},
	})
	if err == nil {
		t.Fatalf("expected install failure due to blocking destination file")
	}

	// Verify rollback cleaned up skill-ok from test-agent
	okPath := filepath.Join(targetDest, "skill-ok")
	if _, err := os.Stat(okPath); !os.IsNotExist(err) {
		t.Errorf("expected skill-ok to be rolled back, but it exists at %s", okPath)
	}

	// Verify lockfile was NOT modified
	lockfile, _ := ReadLockfile()
	if _, ok := lockfile.Namespaces["ns-fail"]; ok {
		t.Errorf("ns-fail should not exist in lockfile after rollback")
	}
}

func TestEngine_Remove_SyncAllDowngrade(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-syncall")
	createLocalTestSkill(t, srcDir, "s1", "# s1")
	createLocalTestSkill(t, srcDir, "s2", "# s2")
	createLocalTestSkill(t, srcDir, "s3", "# s3")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "src-syncall",
		Skills: map[string]models.SkillEntry{
			"s1": {Path: "s1"},
			"s2": {Path: "s2"},
			"s3": {Path: "s3"},
		},
	}
	_ = WriteManifest(srcDir, manifest)

	engine := NewEngine()

	// Install with SyncAll
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-syncall",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		SyncAll:       true,
	})
	if err != nil {
		t.Fatalf("Install SyncAll failed: %v", err)
	}

	// Verify all 3 are installed
	lockfile, _ := ReadLockfile()
	if !lockfile.Namespaces["src-syncall"].SyncAll {
		t.Fatalf("expected SyncAll: true in lockfile")
	}

	// Remove s2 without exclude
	res, err := engine.Remove(context.Background(), RemoveSpec{
		NamespaceName: "src-syncall",
		RemoveSkills:  []string{"s2"},
		Exclude:       false,
	})
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if len(res.Pruned) != 1 || res.Pruned[0].SkillName != "s2" {
		t.Errorf("expected s2 to be pruned, got %v", res.Pruned)
	}

	// Check lockfile: SyncAll must be false, ExplicitSkills must be s1 and s3
	lockfile, _ = ReadLockfile()
	ns := lockfile.Namespaces["src-syncall"]
	if ns.SyncAll {
		t.Errorf("expected SyncAll to be false after downgrade")
	}
	assertEqualSorted(t, "ExplicitSkills", ns.ExplicitSkills, []string{"s1", "s3"})
	if _, ok := ns.Skills.Installed["s2"]; ok {
		t.Errorf("s2 should not be in Installed")
	}
	if _, ok := ns.Skills.Installed["s1"]; !ok {
		t.Errorf("s1 should remain in Installed")
	}

	// Verify filesystem
	if _, err := os.Stat(filepath.Join(targetDest, "s2", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("s2 should be deleted from filesystem")
	}
	if _, err := os.Stat(filepath.Join(targetDest, "s1", "SKILL.md")); err != nil {
		t.Errorf("s1 should remain on filesystem: %v", err)
	}
}

func TestEngine_Remove_ExcludeMode(t *testing.T) {
	tempHome, _ := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-excl")
	createLocalTestSkill(t, srcDir, "s1", "# s1")
	createLocalTestSkill(t, srcDir, "s2", "# s2")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "src-excl",
		Skills: map[string]models.SkillEntry{
			"s1": {Path: "s1"},
			"s2": {Path: "s2"},
		},
	}
	_ = WriteManifest(srcDir, manifest)

	engine := NewEngine()
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-excl",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		SyncAll:       true,
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Remove s1 with Exclude = true
	_, err = engine.Remove(context.Background(), RemoveSpec{
		NamespaceName: "src-excl",
		RemoveSkills:  []string{"s1"},
		Exclude:       true,
	})
	if err != nil {
		t.Fatalf("Remove with exclude failed: %v", err)
	}

	lockfile, _ := ReadLockfile()
	ns := lockfile.Namespaces["src-excl"]
	if !ns.SyncAll {
		t.Errorf("SyncAll should remain true when Exclude is true")
	}
	assertEqualSorted(t, "Excluded", ns.Excluded, []string{"s1"})
}

func TestEngine_Remove_EmptyNamespaceCleanup(t *testing.T) {
	tempHome, _ := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-empty")
	createLocalTestSkill(t, srcDir, "only-skill", "# only")

	engine := NewEngine()
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-empty",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		AddSkills:     []string{"only-skill"},
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	emptyCallbackCalled := false
	removedCallbackCalled := false

	_, err = engine.Remove(context.Background(), RemoveSpec{
		NamespaceName: "src-empty",
		RemoveSkills:  []string{"only-skill"},
		OnNamespaceEmpty: func(ns string) (bool, error) {
			emptyCallbackCalled = true
			return true, nil
		},
		OnRemovedSource: func(ns string) {
			removedCallbackCalled = true
		},
	})
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if !emptyCallbackCalled {
		t.Errorf("expected OnNamespaceEmpty callback to be called")
	}
	if !removedCallbackCalled {
		t.Errorf("expected OnRemovedSource callback to be called")
	}

	lockfile, _ := ReadLockfile()
	if _, ok := lockfile.Namespaces["src-empty"]; ok {
		t.Errorf("src-empty should be removed from lockfile")
	}
}

func TestEngine_Update(t *testing.T) {
	tempHome, _ := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-update")
	createLocalTestSkill(t, srcDir, "up-skill", "# version 1")

	engine := NewEngine()
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-update",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		AddSkills:     []string{"up-skill"},
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// 1. Update when up-to-date
	res, err := engine.Update(context.Background(), UpdateSpec{
		NamespaceName: "src-update",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if res.TotalUnchanged != 1 {
		t.Errorf("expected TotalUnchanged = 1, got %d", res.TotalUnchanged)
	}

	// 2. Update when content changes
	lockfile, _ := ReadLockfile()
	entry := lockfile.Namespaces["src-update"]
	entry.Ref = "old-ref"
	lockfile.Namespaces["src-update"] = entry
	_ = WriteLockfile(lockfile)

	res, err = engine.Update(context.Background(), UpdateSpec{
		NamespaceName: "src-update",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if res.TotalUpdated != 1 {
		t.Errorf("expected TotalUpdated = 1, got %d", res.TotalUpdated)
	}

	// 3. Update non-existent namespace
	_, err = engine.Update(context.Background(), UpdateSpec{
		NamespaceName: "non-existent-ns",
	})
	if err == nil {
		t.Errorf("expected error for non-existent namespace")
	}
}

func TestEngine_Update_ConflictDoesNotDeleteSkill(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)

	srcDir1 := filepath.Join(tempHome, "src-owner")
	createLocalTestSkill(t, srcDir1, "skill-a", "# version 1 owner")

	srcDir2 := filepath.Join(tempHome, "src-other")
	createLocalTestSkill(t, srcDir2, "skill-a", "# version 2 other")

	engine := NewEngine()

	// 1. Install ns-owner with skill-a
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "ns-owner",
		SourceURL:     srcDir1,
		Targets:       []string{"test-agent"},
		AddSkills:     []string{"skill-a"},
	})
	if err != nil {
		t.Fatalf("Install ns-owner failed: %v", err)
	}

	skillFile := filepath.Join(targetDest, "skill-a", "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Fatalf("Expected skill-a to exist after install")
	}

	// 2. ns-other takes ownership with overwrite
	_, err = engine.Install(context.Background(), InstallSpec{
		NamespaceName:    "ns-other",
		SourceURL:        srcDir2,
		Targets:          []string{"test-agent"},
		AddSkills:        []string{"skill-a"},
		ConflictStrategy: "overwrite",
	})
	if err != nil {
		t.Fatalf("Install ns-other failed: %v", err)
	}

	// 3. Update ns-owner with ConflictStrategy "keep" (causing conflict on skill-a)
	lockfile, _ := ReadLockfile()
	entry := lockfile.Namespaces["ns-owner"]
	entry.Ref = "old-ref"
	lockfile.Namespaces["ns-owner"] = entry
	_ = WriteLockfile(lockfile)

	res, err := engine.Update(context.Background(), UpdateSpec{
		NamespaceName:    "ns-owner",
		ConflictStrategy: "keep",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	_ = res

	// 4. Verify skill-a on disk was NOT deleted by PruneSkills
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("CRITICAL BUG: skill-a was deleted from disk during update: %v", err)
	}
	if !strings.Contains(string(content), "version 2 other") {
		t.Errorf("expected skill-a to contain 'version 2 other', got: %s", string(content))
	}
}

func TestEngine_Update_MultiTargetConflictDoesNotDeleteSkill(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("AXEN_TEST_HOME", tempHome)
	resolvers.ResetTargetPathCache()

	axenDir := resolvers.GetAxenDir()
	_ = os.MkdirAll(axenDir, 0755)

	targetDest1 := filepath.Join(tempHome, "targets", "agent-1")
	targetDest2 := filepath.Join(tempHome, "targets", "agent-2")
	_ = os.MkdirAll(targetDest1, 0755)
	_ = os.MkdirAll(targetDest2, 0755)

	cfg := &models.Config{
		Targets: map[string]string{
			"agent-1": targetDest1,
			"agent-2": targetDest2,
		},
	}
	if err := WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}
	resolvers.ResetTargetPathCache()

	lockfile := models.NewLockfile()
	if err := WriteLockfile(lockfile); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	srcDir1 := filepath.Join(tempHome, "src-owner")
	createLocalTestSkill(t, srcDir1, "skill-a", "# version 1 owner")

	srcDir2 := filepath.Join(tempHome, "src-other")
	createLocalTestSkill(t, srcDir2, "skill-a", "# version 2 other")

	engine := NewEngine()

	// 1. Install ns-owner into both agent-1 and agent-2
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "ns-owner",
		SourceURL:     srcDir1,
		Targets:       []string{"agent-1", "agent-2"},
		AddSkills:     []string{"skill-a"},
	})
	if err != nil {
		t.Fatalf("Install ns-owner failed: %v", err)
	}

	// 2. Install ns-other into agent-2 with overwrite
	_, err = engine.Install(context.Background(), InstallSpec{
		NamespaceName:    "ns-other",
		SourceURL:        srcDir2,
		Targets:          []string{"agent-2"},
		AddSkills:        []string{"skill-a"},
		ConflictStrategy: "overwrite",
	})
	if err != nil {
		t.Fatalf("Install ns-other failed: %v", err)
	}

	// 3. Update ns-owner with ConflictStrategy "keep"
	lockfile, _ = ReadLockfile()
	entry := lockfile.Namespaces["ns-owner"]
	entry.Ref = "old-ref"
	lockfile.Namespaces["ns-owner"] = entry
	_ = WriteLockfile(lockfile)

	res, err := engine.Update(context.Background(), UpdateSpec{
		NamespaceName:    "ns-owner",
		ConflictStrategy: "keep",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	_ = res

	// 4. Verify skill-a on agent-1 was kept/updated and NOT deleted
	skill1File := filepath.Join(targetDest1, "skill-a", "SKILL.md")
	content1, err := os.ReadFile(skill1File)
	if err != nil {
		t.Fatalf("CRITICAL BUG: skill-a in agent-1 was deleted during update: %v", err)
	}
	if !strings.Contains(string(content1), "version 1 owner") {
		t.Errorf("expected skill-a in agent-1 to contain 'version 1 owner', got: %s", string(content1))
	}

	// 5. Verify skill-a on agent-2 was NOT deleted by PruneSkills
	skill2File := filepath.Join(targetDest2, "skill-a", "SKILL.md")
	content2, err := os.ReadFile(skill2File)
	if err != nil {
		t.Fatalf("CRITICAL BUG: skill-a in agent-2 was deleted during update: %v", err)
	}
	if !strings.Contains(string(content2), "version 2 other") {
		t.Errorf("expected skill-a in agent-2 to contain 'version 2 other', got: %s", string(content2))
	}
}

func TestEngine_Remove_DeletedSource(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-deleted")
	createLocalTestSkill(t, srcDir, "skill1", "# s1")
	createLocalTestSkill(t, srcDir, "skill2", "# s2")

	engine := NewEngine()
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-deleted",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		AddSkills:     []string{"skill1", "skill2"},
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Delete the source directory to simulate deleted/offline source repository
	if err := os.RemoveAll(srcDir); err != nil {
		t.Fatalf("failed to delete source dir: %v", err)
	}

	// Partial remove when source repo is deleted on disk
	res, err := engine.Remove(context.Background(), RemoveSpec{
		NamespaceName: "src-deleted",
		RemoveSkills:  []string{"skill1"},
	})
	if err != nil {
		t.Fatalf("Partial remove failed when source repo deleted: %v", err)
	}
	if len(res.Pruned) != 1 || res.Pruned[0].SkillName != "skill1" {
		t.Errorf("expected skill1 to be pruned, got %v", res.Pruned)
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill1", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("skill1 should be uninstalled from filesystem")
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill2", "SKILL.md")); err != nil {
		t.Errorf("skill2 should remain on filesystem")
	}

	// Remove all when source repo is deleted on disk
	resAll, err := engine.Remove(context.Background(), RemoveSpec{
		NamespaceName: "src-deleted",
		RemoveAll:     true,
	})
	if err != nil {
		t.Fatalf("RemoveAll failed when source repo deleted: %v", err)
	}
	if len(resAll.Pruned) != 1 || resAll.Pruned[0].SkillName != "skill2" {
		t.Errorf("expected skill2 to be pruned, got %v", resAll.Pruned)
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill2", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("skill2 should be uninstalled from filesystem")
	}

	lockfile, _ := ReadLockfile()
	if _, ok := lockfile.Namespaces["src-deleted"]; ok {
		t.Errorf("src-deleted should be removed from lockfile")
	}
}

func TestEngine_Update_PreservesExplicitSkillsWithoutDefaultBundle(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-update-bundle")
	createLocalTestSkill(t, srcDir, "skill-explicit", "# explicit version 1")
	createLocalTestSkill(t, srcDir, "skill-in-bundle", "# in bundle")

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "src-update-bundle",
		Skills: map[string]models.SkillEntry{
			"skill-explicit":  {Path: "skill-explicit"},
			"skill-in-bundle": {Path: "skill-in-bundle"},
		},
		Bundles: map[string]models.Bundle{
			"standard": {IsDefault: true, Skills: []string{"skill-in-bundle"}},
		},
	}
	_ = WriteManifest(srcDir, manifest)

	engine := NewEngine()

	// Install ONLY explicit skill
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName:  "src-update-bundle",
		SourceURL:      srcDir,
		Targets:        []string{"test-agent"},
		AddSkills:      []string{"skill-explicit"},
		SkipAutoDetect: true,
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify only skill-explicit is installed and no bundles registered
	lockfile, _ := ReadLockfile()
	ns := lockfile.Namespaces["src-update-bundle"]
	if len(ns.Bundles) > 0 {
		t.Fatalf("expected 0 bundles before update, got %v", ns.Bundles)
	}
	if _, ok := ns.Skills.Installed["skill-explicit"]; !ok {
		t.Fatalf("expected skill-explicit in lockfile installed")
	}

	// Change ref to trigger update
	ns.Ref = "outdated-ref"
	lockfile.Namespaces["src-update-bundle"] = ns
	_ = WriteLockfile(lockfile)

	// Run Update
	updateDoneCalled := false
	res, err := engine.Update(context.Background(), UpdateSpec{
		NamespaceName: "src-update-bundle",
		OnUpdateDone: func(ns string, err error) {
			if err == nil {
				updateDoneCalled = true
			}
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !updateDoneCalled {
		t.Errorf("expected OnUpdateDone to be called on successful update")
	}
	if res.TotalUpdated != 1 {
		t.Errorf("expected TotalUpdated = 1, got %d", res.TotalUpdated)
	}

	// Verify lockfile still has NO bundles and standard default bundle was NOT injected
	lockfile, _ = ReadLockfile()
	ns = lockfile.Namespaces["src-update-bundle"]
	if len(ns.Bundles) > 0 {
		t.Errorf("Update injected default bundle into existing explicit intent: %v", ns.Bundles)
	}
	if _, ok := ns.Skills.Installed["skill-explicit"]; !ok {
		t.Errorf("skill-explicit should still be installed")
	}
	if _, ok := ns.Skills.Installed["skill-in-bundle"]; ok {
		t.Errorf("skill-in-bundle should NOT have been installed by update")
	}

	// Verify filesystem
	if _, err := os.Stat(filepath.Join(targetDest, "skill-explicit", "SKILL.md")); err != nil {
		t.Errorf("skill-explicit missing from destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill-in-bundle", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("skill-in-bundle should not exist on filesystem")
	}
}

func TestEngine_Install_SyncAllWithSkillScopeDoesNotPrune(t *testing.T) {
	tempHome, targetDest := setupEngineTestEnv(t)
	srcDir := filepath.Join(tempHome, "src-syncscope")
	createLocalTestSkill(t, srcDir, "skill1", "# skill 1")
	createLocalTestSkill(t, srcDir, "skill2", "# skill 2")

	engine := NewEngine()

	// Initial install with SyncAll
	_, err := engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-syncscope",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		SyncAll:       true,
	})
	if err != nil {
		t.Fatalf("Install SyncAll failed: %v", err)
	}

	// Subsequent install specifying SyncAll along with a SkillScope (e.g. CLI passing --all --skills=skill1)
	_, err = engine.Install(context.Background(), InstallSpec{
		NamespaceName: "src-syncscope",
		SourceURL:     srcDir,
		Targets:       []string{"test-agent"},
		SyncAll:       true,
		SkillScope:    []string{"skill1"},
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Both skills must still exist on filesystem and in lockfile
	if _, err := os.Stat(filepath.Join(targetDest, "skill1", "SKILL.md")); err != nil {
		t.Errorf("skill1 should exist")
	}
	if _, err := os.Stat(filepath.Join(targetDest, "skill2", "SKILL.md")); err != nil {
		t.Errorf("skill2 should exist and not have been pruned: %v", err)
	}

	lockfile, _ := ReadLockfile()
	ns := lockfile.Namespaces["src-syncscope"]
	if len(ns.Skills.Installed) != 2 {
		t.Errorf("expected 2 skills installed in lockfile, got %d", len(ns.Skills.Installed))
	}
}

func TestEngine_SourceOperations(t *testing.T) {
	setupEngineTestEnv(t)
	engine := NewEngine()
	ctx := context.Background()

	// 1. Initially empty
	sources, err := engine.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(sources))
	}

	// 2. AddSource
	err = engine.AddSource(ctx, SourceAddRequest{
		NamespaceName: "alpha-ns",
		SourceURL:     "https://github.com/example/alpha",
		SourceType:    "git",
		UpdatePolicy:  "daily",
		Targets:       []string{"test-agent"},
	})
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	// 3. ListSources returns alpha-ns
	sources, err = engine.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Namespace != "alpha-ns" || sources[0].UpdatePolicy != "daily" {
		t.Errorf("unexpected source info: %+v", sources[0])
	}

	// 4. SetSourcePolicy
	err = engine.SetSourcePolicy(ctx, "alpha-ns", "weekly")
	if err != nil {
		t.Fatalf("SetSourcePolicy failed: %v", err)
	}

	// 5. Verify policy updated via ListSources
	sources, err = engine.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if sources[0].UpdatePolicy != "weekly" {
		t.Errorf("expected weekly policy, got %s", sources[0].UpdatePolicy)
	}
}
