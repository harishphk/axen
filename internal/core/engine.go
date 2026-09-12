package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
	"github.com/harishphk/axen/internal/utils"
)

// ---------------------------------------------------------------------------
// Engine Interface & Types
// ---------------------------------------------------------------------------

// Engine defines the primary deep interface for reconciliation, inspection,
// installation, removal, and updates across skill sources and agent targets.
type Engine interface {
	// Inspect fetches or resolves the source and returns its manifest and available skills/bundles.
	// Powers interactive skill selection in the CLI without redundant re-fetching.
	Inspect(ctx context.Context, sourceURL string, namespace string, opts InspectOptions) (*SourceManifest, error)

	// Install reconciles declared skills/bundles into active agent targets.
	// Encapsulates default bundle auto-detection, conflict resolution, atomic staging, and rollback.
	Install(ctx context.Context, spec InstallSpec) (*ReconcileResult, error)

	// Remove uninstalls skills or entire namespaces.
	// Encapsulates SyncAll downgrade rules, skill pruning, and empty namespace cleanup.
	Remove(ctx context.Context, spec RemoveSpec) (*ReconcileResult, error)

	// Update checks for latest refs and updates installed skills for one or all namespaces.
	Update(ctx context.Context, spec UpdateSpec) (*UpdateResult, error)

	// ListSources returns summary information for all registered sources in the lockfile.
	ListSources(ctx context.Context) ([]SourceInfo, error)

	// AddSource registers a new source namespace in the lockfile and updates the sources cache.
	AddSource(ctx context.Context, req SourceAddRequest) error

	// SetSourcePolicy updates the auto-update policy for an existing registered source namespace.
	SetSourcePolicy(ctx context.Context, namespace string, policy string) error
}

// DefaultEngine is the production implementation of Engine.
type DefaultEngine struct{}

// NewEngine constructs a new DefaultEngine instance.
func NewEngine() Engine {
	return &DefaultEngine{}
}

// InspectOptions provides callbacks for reporting fetch/inspection progress.
type InspectOptions struct {
	OnFetchStart func(namespace string)
	OnFetchDone  func(namespace string, err error)
}

// SourceManifest contains the resolved fetch metadata and manifest for a source.
type SourceManifest struct {
	FetchResult *sources.FetchResult
	Manifest    *models.Manifest
}

// InstallSpec specifies the intent and operational parameters for an install/reconcile action.
type InstallSpec struct {
	NamespaceName             string
	SourceURL                 string
	SourceManifest            *SourceManifest
	AddSkills                 []string
	AddBundles                []string
	SyncAll                   bool
	Targets                   []string
	SkillScope                []string
	TargetScope               []string
	UpdateMode                bool
	Force                     bool
	DryRun                    bool
	SkipAutoDetect            bool
	OnDefaultBundleDetected   func(bundleName string)
	ConflictStrategy          string
	ConflictResolver          func(skillName string, candidates []ConflictCandidate) (string, error)
	UntrackedConflictResolver func(conflicts []UntrackedConflict) (map[string]bool, error)
}

// RemoveSpec specifies the parameters for removing skills or an entire source namespace.
type RemoveSpec struct {
	NamespaceName    string
	RemoveAll        bool
	RemoveSkills     []string
	RemoveBundles    []string
	Exclude          bool
	PromptExclude    func(namespace string) (bool, error)
	OnNamespaceEmpty func(namespace string) (bool, error)
	OnRemovedSource  func(namespace string)
	DryRun           bool
	ConflictStrategy string
	ConflictResolver func(skillName string, candidates []ConflictCandidate) (string, error)
}

// UpdateSpec specifies the parameters for updating installed skills.
type UpdateSpec struct {
	NamespaceName    string // empty = update all registered namespaces
	DryRun           bool
	ConflictStrategy string
	ConflictResolver func(skillName string, candidates []ConflictCandidate) (string, error)
	OnUpdateStart    func(namespace string)
	OnUpdateDone     func(namespace string, err error)
	OnAlreadyUpdated func(namespace string)
	OnUpdated        func(namespace string, installedCount int, prunedCount int)
}

// NamespaceUpdateResult records the update outcome for a single namespace.
type NamespaceUpdateResult struct {
	Namespace      string
	Ref            string
	PreviousRef    string
	InstalledCount int
	PrunedCount    int
	UpToDate       bool
	Error          error
}

// UpdateResult aggregates the results of updating one or more namespaces.
type UpdateResult struct {
	TotalUpdated   int
	TotalUnchanged int
	FailedCount    int
	Results        []NamespaceUpdateResult
	Warnings       []string
}

// ---------------------------------------------------------------------------
// Fetch & Inspection
// ---------------------------------------------------------------------------

// FetchAndResolve handles the flow of fetching a source repository,
// checking for an existing manifest, and generating one if it doesn't exist.
func FetchAndResolve(ctx context.Context, sourceURL, namespace string) (*sources.FetchResult, *models.Manifest, error) {
	fetchResult, err := sources.FetchSource(ctx, sourceURL, namespace)
	if err != nil {
		return nil, nil, err
	}

	if HasManifest(fetchResult.LocalPath) {
		manifest, err := ReadManifest(fetchResult.LocalPath)
		return fetchResult, manifest, err
	}

	scanned, err := ScanSkills(fetchResult.LocalPath)
	if err != nil {
		return nil, nil, err
	}

	manifest := GenerateManifest(namespace, scanned, nil)
	return fetchResult, manifest, nil
}

// Inspect fetches or resolves a source, triggering progress callbacks if provided.
func (e *DefaultEngine) Inspect(ctx context.Context, sourceURL string, namespace string, opts InspectOptions) (*SourceManifest, error) {
	if opts.OnFetchStart != nil {
		opts.OnFetchStart(namespace)
	}

	fetchResult, manifest, err := FetchAndResolve(ctx, sourceURL, namespace)
	if opts.OnFetchDone != nil {
		opts.OnFetchDone(namespace, err)
	}

	if err != nil {
		return nil, err
	}

	return &SourceManifest{
		FetchResult: fetchResult,
		Manifest:    manifest,
	}, nil
}

// ---------------------------------------------------------------------------
// Reconciliation Engine Implementations
// ---------------------------------------------------------------------------

// Install reconciles declared skills/bundles into active agent targets.
func (e *DefaultEngine) Install(ctx context.Context, spec InstallSpec) (*ReconcileResult, error) {
	lockfile, err := ReadLockfile()
	if err != nil {
		return nil, err
	}

	sourceURL := spec.SourceURL
	if sourceURL == "" {
		if ns, ok := lockfile.Namespaces[spec.NamespaceName]; ok && ns.Source != "" {
			sourceURL = ns.Source
		} else {
			sourceURL = spec.NamespaceName
		}
	}

	var fetchResult *sources.FetchResult
	var manifest *models.Manifest

	if spec.SourceManifest != nil && spec.SourceManifest.FetchResult != nil && spec.SourceManifest.Manifest != nil {
		fetchResult = spec.SourceManifest.FetchResult
		manifest = spec.SourceManifest.Manifest
	} else {
		var fetchErr error
		fetchResult, manifest, fetchErr = FetchAndResolve(ctx, sourceURL, spec.NamespaceName)
		if fetchErr != nil {
			return nil, fetchErr
		}
	}

	currentIntent := GetIntent(lockfile, spec.NamespaceName)
	addBundles := copyStrings(spec.AddBundles)
	addSkills := copyStrings(spec.AddSkills)
	skillScope := copyStrings(spec.SkillScope)

	// Auto-detect default bundle if no explicit bundles, skills, or syncAll declared (and not in update mode)
	if !spec.UpdateMode && len(addBundles) == 0 && len(addSkills) == 0 && !spec.SyncAll && !spec.SkipAutoDetect && manifest != nil {
		for name, b := range manifest.Bundles {
			if b.IsDefault {
				if spec.OnDefaultBundleDetected != nil {
					spec.OnDefaultBundleDetected(name)
				}
				addBundles = []string{name}
				if len(skillScope) == 0 {
					skillScope = append(skillScope, b.Skills...)
				}
				break
			}
		}
	}

	// Expand bundle skills into skillScope if skillScope not explicitly passed and not syncAll
	if len(skillScope) == 0 && !spec.SyncAll && manifest != nil {
		for _, bName := range addBundles {
			if b, ok := manifest.Bundles[bName]; ok {
				skillScope = append(skillScope, b.Skills...)
			}
		}
		skillScope = append(skillScope, addSkills...)
	}

	action := IntentAction{
		AddBundles: addBundles,
		AddSkills:  addSkills,
	}
	if spec.SyncAll {
		syncAll := true
		action.SetSyncAll = &syncAll
	}
	if len(spec.Targets) > 0 {
		action.SetTargets = spec.Targets
	}

	newIntent := MergeIntent(currentIntent, action)

	// If SyncAll was requested in this operation, do not restrict reconciliation to a partial skillScope
	effectiveSkillScope := skillScope
	if spec.SyncAll {
		effectiveSkillScope = nil
	}

	reconcileOpts := ReconcileOpts{
		Force:                     spec.Force,
		DryRun:                    spec.DryRun,
		UpdateMode:                spec.UpdateMode || spec.SyncAll,
		SkillScope:                effectiveSkillScope,
		TargetScope:               spec.TargetScope,
		ConflictStrategy:          spec.ConflictStrategy,
		ConflictResolver:          spec.ConflictResolver,
		UntrackedConflictResolver: spec.UntrackedConflictResolver,
	}

	return Reconcile(ctx, lockfile, spec.NamespaceName, sourceURL, fetchResult, manifest, newIntent, reconcileOpts)
}

// Remove uninstalls skills or entire namespaces, managing SyncAll downgrades and cleanup.
func (e *DefaultEngine) Remove(ctx context.Context, spec RemoveSpec) (*ReconcileResult, error) {
	lockfile, err := ReadLockfile()
	if err != nil {
		return nil, err
	}

	nsEntry, exists := lockfile.Namespaces[spec.NamespaceName]
	if !exists {
		return nil, utils.NewAxenError("namespace not found", "INVALID_NAMESPACE")
	}

	skillsToRemove := copyStrings(spec.RemoveSkills)
	bundlesToRemove := copyStrings(spec.RemoveBundles)

	if spec.RemoveAll {
		skillsToRemove = nil
		for s := range nsEntry.Skills.Installed {
			skillsToRemove = append(skillsToRemove, s)
		}
		sort.Strings(skillsToRemove)
		bundlesToRemove = copyStrings(nsEntry.Bundles)
	}

	var fetchResult *sources.FetchResult
	var manifest *models.Manifest
	if !spec.RemoveAll {
		var fetchErr error
		fetchResult, manifest, fetchErr = FetchAndResolve(ctx, nsEntry.Source, spec.NamespaceName)
		if fetchErr != nil {
			utils.Debug("FetchAndResolve failed during remove: %v; proceeding with lockfile state", fetchErr)
		}
	}

	currentIntent := GetIntent(lockfile, spec.NamespaceName)
	isPartialRemove := !spec.RemoveAll && (len(skillsToRemove) > 0 || len(bundlesToRemove) > 0)
	excludeMode := spec.Exclude
	var setSyncAll *bool
	var addSkills []string
	var addExcluded []string

	// Handle SyncAll downgrade invariant when removing individual skills
	if currentIntent.SyncAll && isPartialRemove && !spec.DryRun && !excludeMode {
		if len(skillsToRemove) > 0 && spec.PromptExclude != nil {
			promptExclude, err := spec.PromptExclude(spec.NamespaceName)
			if err != nil {
				return nil, err
			}
			if promptExclude {
				excludeMode = true
			}
		}

		if !excludeMode {
			f := false
			setSyncAll = &f

			remainingBundles := make(map[string]bool)
			for _, b := range currentIntent.Bundles {
				remainingBundles[b] = true
			}
			for _, b := range bundlesToRemove {
				delete(remainingBundles, b)
			}

			bundleCoveredSkills := make(map[string]bool)
			if manifest != nil {
				for bName := range remainingBundles {
					if bundle, ok := manifest.Bundles[bName]; ok {
						for _, s := range bundle.Skills {
							bundleCoveredSkills[s] = true
						}
					}
				}
			}

			removeSet := make(map[string]bool)
			for _, r := range skillsToRemove {
				removeSet[r] = true
			}
			if manifest != nil {
				for _, bName := range bundlesToRemove {
					if bundle, ok := manifest.Bundles[bName]; ok {
						for _, s := range bundle.Skills {
							removeSet[s] = true
						}
					}
				}
			}

			for skillName := range nsEntry.Skills.Installed {
				if removeSet[skillName] {
					continue
				}
				if bundleCoveredSkills[skillName] {
					continue
				}
				addSkills = append(addSkills, skillName)
			}
		}
	}

	if spec.RemoveAll {
		f := false
		setSyncAll = &f
	}

	if excludeMode && len(skillsToRemove) > 0 {
		addExcluded = skillsToRemove
	}

	action := IntentAction{
		RemoveBundles: bundlesToRemove,
		RemoveSkills:  skillsToRemove,
		AddSkills:     addSkills,
		AddExcluded:   addExcluded,
		SetSyncAll:    setSyncAll,
	}

	newIntent := MergeIntent(currentIntent, action)

	reconcileOpts := ReconcileOpts{
		DryRun:           spec.DryRun,
		UpdateMode:       false,
		ConflictStrategy: spec.ConflictStrategy,
		ConflictResolver: spec.ConflictResolver,
	}

	result, err := Reconcile(ctx, lockfile, spec.NamespaceName, nsEntry.Source, fetchResult, manifest, newIntent, reconcileOpts)
	if err != nil {
		return nil, err
	}

	if !spec.DryRun {
		// Re-read lockfile to check if namespace has become empty
		lockfile, err = ReadLockfile()
		if err != nil {
			return nil, err
		}
		if ns, ok := lockfile.Namespaces[spec.NamespaceName]; ok && len(ns.Skills.Installed) == 0 {
			removeSource := true
			if spec.OnNamespaceEmpty != nil {
				var promptErr error
				removeSource, promptErr = spec.OnNamespaceEmpty(spec.NamespaceName)
				if promptErr != nil {
					return nil, promptErr
				}
			}
			if removeSource {
				lockfile = RemoveNamespace(lockfile, spec.NamespaceName)
				if err := WriteLockfile(lockfile); err != nil {
					return nil, err
				}

				cache, _ := ReadSourcesIndex()
				if cache != nil {
					delete(cache.Namespaces, spec.NamespaceName)
					if err := WriteSourcesIndex(cache); err != nil {
						return nil, err
					}
				}

				if spec.OnRemovedSource != nil {
					spec.OnRemovedSource(spec.NamespaceName)
				}
			}
		}
	}

	return result, nil
}

// Update checks for latest refs and updates installed skills across namespaces.
func (e *DefaultEngine) Update(ctx context.Context, spec UpdateSpec) (*UpdateResult, error) {
	lockfile, err := ReadLockfile()
	if err != nil {
		return nil, err
	}

	var toUpdate []string
	if spec.NamespaceName != "" {
		if _, ok := lockfile.Namespaces[spec.NamespaceName]; !ok {
			return nil, fmt.Errorf("namespace %q not found in lockfile", spec.NamespaceName)
		}
		toUpdate = append(toUpdate, spec.NamespaceName)
	} else {
		for ns := range lockfile.Namespaces {
			toUpdate = append(toUpdate, ns)
		}
		sort.Strings(toUpdate)
	}

	result := &UpdateResult{}

	for _, nsName := range toUpdate {
		// Re-read fresh lockfile state on each iteration
		currentLockfile, readErr := ReadLockfile()
		if readErr == nil && currentLockfile != nil {
			lockfile = currentLockfile
		}
		nsEntry := lockfile.Namespaces[nsName]

		if spec.OnUpdateStart != nil {
			spec.OnUpdateStart(nsName)
		}

		fetchResult, manifest, err := FetchAndResolve(ctx, nsEntry.Source, nsName)
		if err != nil {
			if spec.OnUpdateDone != nil {
				spec.OnUpdateDone(nsName, err)
			}
			result.FailedCount++
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", nsName, err))
			result.Results = append(result.Results, NamespaceUpdateResult{
				Namespace: nsName,
				Error:     err,
			})
			continue
		}

		if fetchResult.Ref == nsEntry.Ref {
			if spec.OnAlreadyUpdated != nil {
				spec.OnAlreadyUpdated(nsName)
			}
			if spec.OnUpdateDone != nil {
				spec.OnUpdateDone(nsName, nil)
			}
			result.TotalUnchanged++
			result.Results = append(result.Results, NamespaceUpdateResult{
				Namespace:   nsName,
				Ref:         fetchResult.Ref,
				PreviousRef: nsEntry.Ref,
				UpToDate:    true,
			})
			continue
		}

		installSpec := InstallSpec{
			NamespaceName:    nsName,
			SourceURL:        nsEntry.Source,
			SourceManifest:   &SourceManifest{FetchResult: fetchResult, Manifest: manifest},
			UpdateMode:       true,
			SkipAutoDetect:   true,
			Force:            spec.ConflictStrategy == "overwrite",
			DryRun:           spec.DryRun,
			ConflictStrategy: spec.ConflictStrategy,
			ConflictResolver: spec.ConflictResolver,
		}

		res, err := e.Install(ctx, installSpec)
		if err != nil {
			if spec.OnUpdateDone != nil {
				spec.OnUpdateDone(nsName, err)
			}
			result.FailedCount++
			result.Results = append(result.Results, NamespaceUpdateResult{
				Namespace: nsName,
				Error:     err,
			})
			continue
		}

		if spec.OnUpdated != nil {
			spec.OnUpdated(nsName, len(res.Installed), len(res.Pruned))
		}
		if spec.OnUpdateDone != nil {
			spec.OnUpdateDone(nsName, nil)
		}
		result.TotalUpdated += len(res.Installed)
		result.Results = append(result.Results, NamespaceUpdateResult{
			Namespace:      nsName,
			Ref:            fetchResult.Ref,
			PreviousRef:    nsEntry.Ref,
			InstalledCount: len(res.Installed),
			PrunedCount:    len(res.Pruned),
		})
	}

	if result.FailedCount > 0 {
		return result, fmt.Errorf("%d namespace(s) failed to update", result.FailedCount)
	}

	return result, nil
}

// ListSources returns summary information for all registered sources in the lockfile.
func (e *DefaultEngine) ListSources(ctx context.Context) ([]SourceInfo, error) {
	return ListSources()
}

// AddSource registers a new source namespace in the lockfile and updates the sources cache.
func (e *DefaultEngine) AddSource(ctx context.Context, req SourceAddRequest) error {
	return AddSource(req)
}

// SetSourcePolicy updates the auto-update policy for an existing registered source namespace.
func (e *DefaultEngine) SetSourcePolicy(ctx context.Context, namespace string, policy string) error {
	return SetSourcePolicy(namespace, policy)
}

// ---------------------------------------------------------------------------
// Intent & State Types
// ---------------------------------------------------------------------------

// Intent captures what the user declared — NOT what skills to install.
// This is the "what", not the "how".
type Intent struct {
	Bundles        []string // subscribed bundle names
	ExplicitSkills []string // individually requested skills
	Excluded       []string // explicitly excluded skills
	Targets        []string // target destinations
	SyncAll        bool     // install everything from this source
}

// IntentAction describes what the user asked to change in this session.
type IntentAction struct {
	AddBundles    []string
	RemoveBundles []string
	AddSkills     []string
	RemoveSkills  []string
	AddExcluded   []string
	SetTargets    []string // nil = keep existing
	SetSyncAll    *bool    // nil = keep existing
}

// ResolvedState is the concrete desired state after expanding
// bundles against the manifest.
type ResolvedState struct {
	DesiredSkills map[string]bool // every skill name that should exist
	Intent        Intent          // preserved for lockfile persistence
}

// Plan is the diff between current installed state and desired state.
type Plan struct {
	ToInstall []string // skills in desired but NOT currently installed
	ToPrune   []string // skills currently installed but NOT in desired
	Unchanged []string // skills in BOTH current and desired
}

// ReconcileOpts controls how the reconciliation engine operates.
type ReconcileOpts struct {
	Force                     bool
	DryRun                    bool
	UpdateMode                bool     // true = re-install unchanged skills too (for update command)
	SkillScope                []string // if set, only process these specific skills (for install with explicit flags)
	TargetScope               []string // if set, temporary targets scope for this run only
	ConflictStrategy          string
	ConflictResolver          func(skillName string, candidates []ConflictCandidate) (string, error)
	UntrackedConflictResolver func(conflicts []UntrackedConflict) (map[string]bool, error)
}

// ReconcileResult contains everything the CLI needs to display results.
type ReconcileResult struct {
	Installed []InstallResult
	Pruned    []PrunedSkill
	Skipped   []InstallResult
	Conflicts []InstallResult
	Plan      Plan
}

// ---------------------------------------------------------------------------
// Pure Functions (no I/O, independently testable)
// ---------------------------------------------------------------------------

// GetIntent extracts the declared Intent from a lockfile namespace entry.
func GetIntent(lockfile *models.Lockfile, namespace string) Intent {
	ns, ok := lockfile.Namespaces[namespace]
	if !ok {
		return Intent{}
	}
	return Intent{
		Bundles:        ns.Bundles,
		ExplicitSkills: ns.ExplicitSkills,
		Excluded:       ns.Excluded,
		Targets:        ns.Targets,
		SyncAll:        ns.SyncAll,
	}
}

// MergeIntent merges a user action into the current intent, producing a new intent.
// This is a pure function — no side effects.
func MergeIntent(current Intent, action IntentAction) Intent {
	result := Intent{
		Bundles:        copyStrings(current.Bundles),
		ExplicitSkills: copyStrings(current.ExplicitSkills),
		Excluded:       copyStrings(current.Excluded),
		Targets:        copyStrings(current.Targets),
		SyncAll:        current.SyncAll,
	}

	// Add bundles (deduplicated)
	for _, b := range action.AddBundles {
		if !containsStr(result.Bundles, b) {
			result.Bundles = append(result.Bundles, b)
		}
	}

	// Remove bundles
	result.Bundles = removeStrs(result.Bundles, action.RemoveBundles)

	// Add explicit skills (deduplicated)
	for _, s := range action.AddSkills {
		if !containsStr(result.ExplicitSkills, s) {
			result.ExplicitSkills = append(result.ExplicitSkills, s)
		}
		// Installing a skill removes it from the excluded list
		result.Excluded = removeStrs(result.Excluded, []string{s})
	}

	// Remove explicit skills
	result.ExplicitSkills = removeStrs(result.ExplicitSkills, action.RemoveSkills)

	// Add to excluded list
	for _, e := range action.AddExcluded {
		if !containsStr(result.Excluded, e) {
			result.Excluded = append(result.Excluded, e)
		}
	}

	// Set targets (nil = keep existing)
	if action.SetTargets != nil {
		result.Targets = action.SetTargets
	}

	// Set sync all (nil = keep existing)
	if action.SetSyncAll != nil {
		result.SyncAll = *action.SetSyncAll
	}

	sort.Strings(result.Bundles)
	sort.Strings(result.ExplicitSkills)
	sort.Strings(result.Excluded)

	return result
}

// ResolveIntent expands an Intent against a manifest to produce
// the concrete set of desired skills.
func ResolveIntent(intent Intent, manifest *models.Manifest) ResolvedState {
	desired := make(map[string]bool)

	if manifest != nil {
		if intent.SyncAll {
			// SyncAll: every skill in the manifest is desired
			for name := range manifest.Skills {
				desired[name] = true
			}
		} else {
			// Expand bundles
			for _, bundleName := range intent.Bundles {
				if bundle, ok := manifest.Bundles[bundleName]; ok {
					for _, skill := range bundle.Skills {
						desired[skill] = true
					}
				}
			}
			// Add explicit skills
			for _, skill := range intent.ExplicitSkills {
				desired[skill] = true
			}
		}
	} else {
		for _, skill := range intent.ExplicitSkills {
			desired[skill] = true
		}
	}

	// Remove excluded
	for _, exc := range intent.Excluded {
		delete(desired, exc)
	}

	return ResolvedState{
		DesiredSkills: desired,
		Intent:        intent,
	}
}

// DiffState compares the currently installed skills against the desired state
// and returns a Plan describing what needs to change.
func DiffState(currentInstalled map[string]models.LockfileSkill, desired ResolvedState) Plan {
	var toInstall, toPrune, unchanged []string

	// Skills in desired but not in current → need to install
	for skill := range desired.DesiredSkills {
		if _, exists := currentInstalled[skill]; exists {
			unchanged = append(unchanged, skill)
		} else {
			toInstall = append(toInstall, skill)
		}
	}

	// Skills in current but not in desired → need to prune
	for skill := range currentInstalled {
		if !desired.DesiredSkills[skill] {
			toPrune = append(toPrune, skill)
		}
	}

	sort.Strings(toInstall)
	sort.Strings(toPrune)
	sort.Strings(unchanged)

	return Plan{
		ToInstall: toInstall,
		ToPrune:   toPrune,
		Unchanged: unchanged,
	}
}

// ---------------------------------------------------------------------------
// Orchestrator (handles I/O)
// ---------------------------------------------------------------------------

// Reconcile is the single entry point for all state changes.
// It resolves the intent, computes a plan, installs/prunes skills,
// and persists the lockfile and sources cache.
func Reconcile(
	ctx context.Context,
	lockfile *models.Lockfile,
	namespace string,
	source string,
	fetchResult *sources.FetchResult,
	manifest *models.Manifest,
	intent Intent,
	opts ReconcileOpts,
) (*ReconcileResult, error) {
	// Read current installed state
	nsEntry := lockfile.Namespaces[namespace]
	currentInstalled := nsEntry.Skills.Installed
	if currentInstalled == nil {
		currentInstalled = make(map[string]models.LockfileSkill)
	}

	// Phase 1: Resolve intent → concrete desired skills
	resolved := ResolveIntent(intent, manifest)

	// Phase 2: Diff current vs desired
	plan := DiffState(currentInstalled, resolved)

	// Phase 3: Determine what to process
	var toProcess []string
	if len(opts.SkillScope) > 0 {
		// Scoped mode: only process specific skills the user asked for.
		// This handles "install --skills=X --targets=Y" where X may already be installed
		// but needs to be re-installed to a new target.
		toProcess = opts.SkillScope
	} else if opts.UpdateMode {
		// Update mode: re-install all desired skills (content may have changed)
		for s := range resolved.DesiredSkills {
			toProcess = append(toProcess, s)
		}
		sort.Strings(toProcess)
	} else {
		// Normal mode: only install truly new skills
		toProcess = append(toProcess, plan.ToInstall...)
	}

	// Phase 4: Install
	var installResults []InstallResult
	if len(toProcess) > 0 {
		activeTargets := intent.Targets
		if len(opts.TargetScope) > 0 {
			activeTargets = opts.TargetScope
		}

		var err error
		installResults, err = InstallSkills(fetchResult.LocalPath, manifest, lockfile, namespace,
			InstallOptions{
				Force:                     opts.Force,
				DryRun:                    opts.DryRun,
				Targets:                   activeTargets,
				SkillFilter:               toProcess,
				ConflictStrategy:          opts.ConflictStrategy,
				ConflictResolver:          opts.ConflictResolver,
				UntrackedConflictResolver: opts.UntrackedConflictResolver,
			})
		if err != nil {
			return nil, err
		}
	}

	// Phase 5: Prune
	var pruneResults []PrunedSkill
	if opts.UpdateMode {
		// Full pruning: compare all old skills against new install results.
		// This also handles partial target pruning (skill stays but loses targets).
		var err error
		pruneResults, err = PruneSkills(currentInstalled, installResults, nsEntry.Targets, opts.DryRun, opts.SkillScope)
		if err != nil {
			return nil, err
		}
	} else {
		// Targeted pruning: only remove skills from plan.ToPrune
		for _, skillName := range plan.ToPrune {
			oldInfo := currentInstalled[skillName]
			targets := oldInfo.Targets
			if len(targets) == 0 {
				targets = intent.Targets
			}
			removed, err := UninstallSkillFromTargets(skillName, targets, opts.DryRun)
			if err != nil {
				continue
			}
			if len(removed) > 0 {
				pruneResults = append(pruneResults, PrunedSkill{SkillName: skillName, Removed: removed})
			}
		}
	}

	// Phase 6: Persist
	if !opts.DryRun {
		if err := persistState(lockfile, namespace, source, fetchResult, manifest, resolved, installResults, pruneResults, currentInstalled); err != nil {
			return nil, fmt.Errorf("failed to persist lockfile: %w", err)
		}
		updateSourcesCache(namespace, manifest)
	}

	// Build result for the caller
	result := &ReconcileResult{Plan: plan}
	for _, r := range installResults {
		switch r.Status {
		case "installed":
			result.Installed = append(result.Installed, r)
		case "skipped":
			result.Skipped = append(result.Skipped, r)
		case "conflict", "untracked_conflict":
			result.Conflicts = append(result.Conflicts, r)
		}
	}
	result.Pruned = pruneResults

	return result, nil
}

// ---------------------------------------------------------------------------
// Persistence (internal)
// ---------------------------------------------------------------------------

// persistState writes the reconciled state to the lockfile.
func persistState(
	lockfile *models.Lockfile,
	namespace string,
	source string,
	fetchResult *sources.FetchResult,
	manifest *models.Manifest,
	resolved ResolvedState,
	installResults []InstallResult,
	pruneResults []PrunedSkill,
	oldInstalled map[string]models.LockfileSkill,
) error {
	newInstalled := make(map[string]models.LockfileSkill)

	// 1. Carry forward old installed skills, excluding pruned ones
	prunedMap := make(map[string]bool)
	for _, p := range pruneResults {
		prunedMap[p.SkillName] = true
	}
	for name, info := range oldInstalled {
		if !prunedMap[name] {
			newInstalled[name] = info
		}
	}

	// 2. Override/add newly installed skills
	defaultTargets := make([]string, len(resolved.Intent.Targets))
	copy(defaultTargets, resolved.Intent.Targets)
	sort.Strings(defaultTargets)

	for _, r := range installResults {
		if r.Status != "installed" {
			continue
		}

		var skillTargets []string
		for _, d := range r.Destinations {
			skillTargets = append(skillTargets, d.Target)
		}
		// Preserve any skipped destinations that were already installed
		if oldSkill, ok := oldInstalled[r.SkillName]; ok {
			oldTgs := oldSkill.Targets
			if len(oldTgs) == 0 {
				oldTgs = resolved.Intent.Targets
			}
			oldMap := make(map[string]bool)
			for _, ot := range oldTgs {
				oldMap[ot] = true
			}
			for _, d := range r.SkippedDestinations {
				if oldMap[d.Target] {
					skillTargets = append(skillTargets, d.Target)
				}
			}
		}
		sort.Strings(skillTargets)

		isOverride := strings.Join(skillTargets, ",") != strings.Join(defaultTargets, ",")

		skillVersion := ""
		if manifest != nil {
			if se, ok := manifest.Skills[r.SkillName]; ok {
				skillVersion = se.Version
			}
		}

		ls := models.LockfileSkill{}
		if isOverride {
			ls.Targets = skillTargets
		}
		if skillVersion != "" {
			ls.Version = skillVersion
		}

		newInstalled[r.SkillName] = ls
	}

	// 3. Build namespace entry with intent + state
	nsSource := source
	if fetchResult != nil && fetchResult.Type == sources.SourceTypeLocal {
		nsSource = fetchResult.LocalPath
	}

	existing, hasExisting := lockfile.Namespaces[namespace]

	nsType := ""
	if fetchResult != nil {
		nsType = string(fetchResult.Type)
	} else if hasExisting {
		nsType = existing.Type
	}

	nsRef := ""
	if fetchResult != nil {
		nsRef = fetchResult.Ref
	} else if hasExisting {
		nsRef = existing.Ref
	}

	updatePolicy := ""
	lastCheckedAt := ""
	if hasExisting {
		updatePolicy = existing.UpdatePolicy
		lastCheckedAt = existing.LastCheckedAt
	}

	entry := models.NamespaceEntry{
		Source:         nsSource,
		Type:           nsType,
		Ref:            nsRef,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		UpdatePolicy:   updatePolicy,
		LastCheckedAt:  lastCheckedAt,
		SyncAll:        resolved.Intent.SyncAll,
		Excluded:       resolved.Intent.Excluded,
		Targets:        resolved.Intent.Targets,
		Bundles:        resolved.Intent.Bundles,
		ExplicitSkills: resolved.Intent.ExplicitSkills,
		Skills: models.NamespaceSkills{
			Installed: newInstalled,
		},
	}

	UpsertNamespace(lockfile, namespace, entry)
	return WriteLockfile(lockfile)
}

// updateSourcesCache refreshes the sources.json cache from the manifest.
func updateSourcesCache(namespace string, manifest *models.Manifest) {
	if manifest == nil {
		return
	}
	cache, _ := ReadSourcesIndex()
	if cache == nil {
		cache = models.NewSourcesCache()
	}
	cacheNs := models.CacheNamespace{Available: make(map[string]models.AvailableSkill)}
	for skillName, entry := range manifest.Skills {
		cacheNs.Available[skillName] = models.AvailableSkill{Path: entry.Path, Version: entry.Version}
	}
	cache.Namespaces[namespace] = cacheNs
	_ = WriteSourcesIndex(cache)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeStrs(slice []string, toRemove []string) []string {
	if len(toRemove) == 0 {
		return slice
	}
	removeMap := make(map[string]bool)
	for _, r := range toRemove {
		removeMap[r] = true
	}
	var result []string
	for _, s := range slice {
		if !removeMap[s] {
			result = append(result, s)
		}
	}
	return result
}
