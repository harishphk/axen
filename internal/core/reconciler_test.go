package core

import (
	"axen/internal/models"
	"axen/internal/sources"
	"testing"
)

func TestReconciler(t *testing.T) {
	lockfile := models.NewLockfile()
	manifest := models.NewManifest("test-ns")
	manifest.Skills["s1"] = models.SkillEntry{Version: "1.0"}

	installResults := []InstallResult{
		{
			SkillName: "s1",
			Status:    "installed",
			Destinations: []Destination{
				{Target: "t1", Path: "path1"},
			},
		},
		{
			SkillName: "s2", // not installed
			Status:    "skipped",
		},
	}

	fetchResult := &sources.FetchResult{
		Type:      sources.SourceTypeLocal,
		LocalPath: "/local/path",
		Ref:       "main",
	}

	options := ReconcileOptions{
		DefaultTargets: []string{"t2"},
		SkillFilter:    []string{"s2"},
		OldSkillsState: map[string]models.LockfileSkill{
			"s3": {Version: "0.5"}, // should be kept
			"s2": {Version: "0.9"}, // should be filtered out
		},
	}

	lockfile = ReconcileLockfile(lockfile, "test-ns", manifest, installResults, fetchResult, "http://source", options)

	ns, ok := lockfile.Namespaces["test-ns"]
	if !ok {
		t.Fatalf("Expected namespace test-ns")
	}

	if ns.Source != "/local/path" {
		t.Fatalf("Expected source /local/path, got %s", ns.Source)
	}

	if ns.Type != string(sources.SourceTypeLocal) {
		t.Fatalf("Expected type local")
	}

	if ns.Ref != "main" {
		t.Fatalf("Expected ref main")
	}

	if len(ns.Targets) != 1 || ns.Targets[0] != "t2" {
		t.Fatalf("Expected default targets [t2], got %v", ns.Targets)
	}

	if len(ns.Skills.Installed) != 2 {
		t.Fatalf("Expected 2 installed skills (s1, s3), got %v", ns.Skills.Installed)
	}

	s1, ok := ns.Skills.Installed["s1"]
	if !ok {
		t.Fatalf("Expected s1")
	}
	if len(s1.Targets) != 1 || s1.Targets[0] != "t1" {
		t.Fatalf("Expected s1 override targets [t1]")
	}
	if s1.Version != "1.0" {
		t.Fatalf("Expected s1 version 1.0")
	}

	s3, ok := ns.Skills.Installed["s3"]
	if !ok {
		t.Fatalf("Expected s3 kept from old state")
	}
	if s3.Version != "0.5" {
		t.Fatalf("Expected s3 version 0.5")
	}

	// Test with non-local source
	fetchResult.Type = sources.SourceTypeGit
	lockfile = ReconcileLockfile(lockfile, "test-ns-git", manifest, nil, fetchResult, "http://source", ReconcileOptions{})
	nsGit := lockfile.Namespaces["test-ns-git"]
	if nsGit.Source != "http://source" {
		t.Fatalf("Expected http://source, got %s", nsGit.Source)
	}
}
