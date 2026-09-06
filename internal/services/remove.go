package services

import (
	"context"
	"fmt"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
)

type RemoveService struct{}

type RemoveRequest struct {
	NamespaceName    string
	SourceURL        string
	FetchResult      *sources.FetchResult
	Manifest         *models.Manifest
	RemoveSkills     []string
	RemoveBundles    []string
	AddSkills        []string
	AddExcluded      []string
	RemoveAll        bool
	SetSyncAll       *bool
	DryRun           bool
	ConflictStrategy string
	ConflictResolver func(skillName string, candidates []core.ConflictCandidate) (string, error)
}

func (s *RemoveService) Remove(ctx context.Context, req RemoveRequest) (*core.ReconcileResult, error) {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return nil, err
	}

	if _, ok := lockfile.Namespaces[req.NamespaceName]; !ok {
		return nil, fmt.Errorf("namespace %q not found in lockfile", req.NamespaceName)
	}

	currentIntent := core.GetIntent(lockfile, req.NamespaceName)
	removeSkills := req.RemoveSkills
	removeBundles := req.RemoveBundles

	if req.RemoveAll {
		if nsEntry, exists := lockfile.Namespaces[req.NamespaceName]; exists {
			for s := range nsEntry.Skills.Installed {
				removeSkills = append(removeSkills, s)
			}
			removeBundles = nsEntry.Bundles
		}
	}

	action := core.IntentAction{
		RemoveBundles: removeBundles,
		RemoveSkills:  removeSkills,
		AddSkills:     req.AddSkills,
		AddExcluded:   req.AddExcluded,
		SetSyncAll:    req.SetSyncAll,
	}

	newIntent := core.MergeIntent(currentIntent, action)

	reconcileOpts := core.ReconcileOpts{
		DryRun:           req.DryRun,
		UpdateMode:       false,
		ConflictStrategy: req.ConflictStrategy,
		ConflictResolver: req.ConflictResolver,
	}

	result, err := core.Reconcile(ctx, lockfile, req.NamespaceName, req.SourceURL, req.FetchResult, req.Manifest, newIntent, reconcileOpts)
	if err != nil {
		return nil, err
	}

	return result, nil
}
