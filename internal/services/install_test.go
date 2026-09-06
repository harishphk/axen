package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
)

func createMockSource(t *testing.T, tmpDir string) (string, *sources.FetchResult, *models.Manifest) {
	t.Helper()

	srcDir := filepath.Join(tmpDir, "mock-source")
	skillDir := filepath.Join(srcDir, "my-skill")
	_ = os.MkdirAll(skillDir, 0755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644)

	manifest := &models.Manifest{
		AxenVersion: "1",
		Name:        "mock-source",
		Skills: map[string]models.SkillEntry{
			"my-skill": {
				Path:    "my-skill",
				Version: "1.0.0",
			},
		},
		Bundles: map[string]models.Bundle{
			"core": {
				Skills: []string{"my-skill"},
			},
		},
	}

	fetchResult := &sources.FetchResult{
		Type:           sources.SourceTypeLocal,
		LocalPath:      srcDir,
		ResolvedSource: srcDir,
		Ref:            "local",
	}

	return srcDir, fetchResult, manifest
}

func TestInstallService_Install(t *testing.T) {
	tmpDir := setupTestEnvironment(t)
	srcDir, fetchResult, manifest := createMockSource(t, tmpDir)

	sourceSvc := &SourceService{}
	err := sourceSvc.Add(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     srcDir,
		SourceType:    "local",
		Ref:           fetchResult.Ref,
		Manifest:      manifest,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	svc := &InstallService{}

	t.Run("Dry Run", func(t *testing.T) {
		req := InstallRequest{
			NamespaceName: "test-ns",
			SourceURL:     srcDir,
			FetchResult:   fetchResult,
			Manifest:      manifest,
			AddSkills:     []string{"my-skill"},
			DryRun:        true,
		}

		res, err := svc.Install(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("Install Skill", func(t *testing.T) {
		req := InstallRequest{
			NamespaceName: "test-ns",
			SourceURL:     srcDir,
			FetchResult:   fetchResult,
			Manifest:      manifest,
			AddSkills:     []string{"my-skill"},
			Force:         true,
		}

		res, err := svc.Install(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Installed) == 0 {
			t.Fatal("expected at least 1 installed skill")
		}

		lockfile, _ := core.ReadLockfile()
		if _, ok := lockfile.Namespaces["test-ns"].Skills.Installed["my-skill"]; !ok {
			t.Errorf("expected skill to be recorded in lockfile")
		}
	})
}
