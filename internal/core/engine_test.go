package core

import (
	"axen/internal/models"
	"reflect"
	"sort"
	"testing"
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
		name             string
		current          map[string]models.LockfileSkill
		desired          map[string]bool
		expectInstall    []string
		expectPrune      []string
		expectUnchanged  []string
	}{
		{
			name:    "fresh install — everything is new",
			current: map[string]models.LockfileSkill{},
			desired: map[string]bool{"a": true, "b": true},
			expectInstall:   []string{"a", "b"},
			expectPrune:     nil,
			expectUnchanged: nil,
		},
		{
			name:    "nothing changed",
			current: map[string]models.LockfileSkill{"a": {}, "b": {}},
			desired: map[string]bool{"a": true, "b": true},
			expectInstall:   nil,
			expectPrune:     nil,
			expectUnchanged: []string{"a", "b"},
		},
		{
			name:    "everything removed",
			current: map[string]models.LockfileSkill{"a": {}, "b": {}},
			desired: map[string]bool{},
			expectInstall:   nil,
			expectPrune:     []string{"a", "b"},
			expectUnchanged: nil,
		},
		{
			name:    "mixed: some new, some existing, some removed",
			current: map[string]models.LockfileSkill{"a": {}, "b": {}, "c": {}},
			desired: map[string]bool{"b": true, "c": true, "d": true},
			expectInstall:   []string{"d"},
			expectPrune:     []string{"a"},
			expectUnchanged: []string{"b", "c"},
		},
		{
			name:    "both empty",
			current: map[string]models.LockfileSkill{},
			desired: map[string]bool{},
			expectInstall:   nil,
			expectPrune:     nil,
			expectUnchanged: nil,
		},
		{
			name:    "nil current",
			current: nil,
			desired: map[string]bool{"a": true},
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
