package services

import (
	"context"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
)

type InstallService struct{}

type FetchOptions struct {
	OnFetchStart func(namespace string)
	OnFetchDone  func(namespace string, err error)
}

func (s *InstallService) FetchManifest(ctx context.Context, sourceURL string, namespaceName string, opts FetchOptions) (*sources.FetchResult, *models.Manifest, error) {
	if opts.OnFetchStart != nil {
		opts.OnFetchStart(namespaceName)
	}

	fetchResult, manifest, err := core.FetchAndResolve(ctx, sourceURL, namespaceName)

	if opts.OnFetchDone != nil {
		opts.OnFetchDone(namespaceName, err)
	}

	if err != nil {
		return nil, nil, err
	}
	return fetchResult, manifest, nil
}

type InstallRequest struct {
	NamespaceName    string
	SourceURL        string
	FetchResult      *sources.FetchResult
	Manifest         *models.Manifest
	AddSkills        []string
	AddBundles       []string
	SetSyncAll       bool
	SetTargets       []string
	SkillScope       []string
	TargetScope      []string
	UpdateMode       bool
	Force            bool
	DryRun           bool
	ConflictStrategy string
	ConflictResolver func(skillName string, candidates []core.ConflictCandidate) (string, error)
	UntrackedResolver func(conflicts []core.UntrackedConflict) (map[string]bool, error)
}

func (s *InstallService) Install(ctx context.Context, req InstallRequest) (*core.ReconcileResult, error) {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return nil, err
	}

	currentIntent := core.GetIntent(lockfile, req.NamespaceName)
	action := core.IntentAction{
		AddBundles: req.AddBundles,
		AddSkills:  req.AddSkills,
	}
	if req.SetSyncAll {
		syncAll := true
		action.SetSyncAll = &syncAll
	}
	if len(req.SetTargets) > 0 {
		action.SetTargets = req.SetTargets
	}

	newIntent := core.MergeIntent(currentIntent, action)

	reconcileOpts := core.ReconcileOpts{
		Force:                     req.Force,
		DryRun:                    req.DryRun,
		UpdateMode:                req.UpdateMode,
		SkillScope:                req.SkillScope,
		TargetScope:               req.TargetScope,
		ConflictStrategy:          req.ConflictStrategy,
		ConflictResolver:          req.ConflictResolver,
		UntrackedConflictResolver: req.UntrackedResolver,
	}

	result, err := core.Reconcile(ctx, lockfile, req.NamespaceName, req.SourceURL, req.FetchResult, req.Manifest, newIntent, reconcileOpts)
	if err != nil {
		return nil, err
	}

	return result, nil
}
