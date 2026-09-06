package core

import (
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/sources"
	"context"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Types
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
		pruneResults, err = PruneSkills(currentInstalled, installResults, nsEntry.Targets, opts.DryRun, nil)
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
		persistState(lockfile, namespace, source, fetchResult, manifest, resolved, installResults, pruneResults, currentInstalled)
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
) {
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
	_ = WriteLockfile(lockfile)
}

// updateSourcesCache refreshes the sources.json cache from the manifest.
func updateSourcesCache(namespace string, manifest *models.Manifest) {
	if manifest == nil {
		return
	}
	cache, _ := ReadSourcesIndex()
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
