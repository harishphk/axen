package services

import (
	"context"
	"testing"

	"github.com/harishphk/axen/internal/core"
)

func TestRemoveService_Remove(t *testing.T) {
	tmpDir := setupTestEnvironment(t)
	srcDir, fetchResult, manifest := createMockSource(t, tmpDir)

	sourceSvc := &SourceService{}
	_ = sourceSvc.Add(SourceAddRequest{
		NamespaceName: "test-ns",
		SourceURL:     srcDir,
		SourceType:    "local",
		Ref:           fetchResult.Ref,
		Manifest:      manifest,
	})

	installSvc := &InstallService{}
	_, err := installSvc.Install(context.Background(), InstallRequest{
		NamespaceName: "test-ns",
		SourceURL:     srcDir,
		FetchResult:   fetchResult,
		Manifest:      manifest,
		AddSkills:     []string{"my-skill"},
		Force:         true,
	})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	removeSvc := &RemoveService{}

	t.Run("Missing Namespace Error", func(t *testing.T) {
		_, err := removeSvc.Remove(context.Background(), RemoveRequest{
			NamespaceName: "non-existent-ns",
		})
		if err == nil {
			t.Fatal("expected error for non-existent namespace")
		}
	})

	t.Run("Remove Specific Skill", func(t *testing.T) {
		res, err := removeSvc.Remove(context.Background(), RemoveRequest{
			NamespaceName: "test-ns",
			SourceURL:     srcDir,
			FetchResult:   fetchResult,
			Manifest:      manifest,
			RemoveSkills:  []string{"my-skill"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Pruned) == 0 {
			t.Fatal("expected at least 1 pruned skill")
		}

		lockfile, _ := core.ReadLockfile()
		if _, ok := lockfile.Namespaces["test-ns"].Skills.Installed["my-skill"]; ok {
			t.Errorf("expected skill to be removed from lockfile")
		}
	})
}
