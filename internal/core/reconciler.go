package core

import (
	"axen/internal/models"
	"axen/internal/sources"
	"sort"
	"strings"
	"time"
)

type ReconcileOptions struct {
	DefaultTargets []string
	SkillFilter    []string
	OldSkillsState map[string]models.LockfileSkill
	OldExcluded    []string
	SyncAll        bool
}

func ReconcileLockfile(
	lockfile *models.Lockfile,
	namespaceName string,
	manifest *models.Manifest,
	installResults []InstallResult,
	fetchResult *sources.FetchResult,
	source string,
	options ReconcileOptions,
) *models.Lockfile {
	var installed []InstallResult
	for _, r := range installResults {
		if r.Status == "installed" {
			installed = append(installed, r)
		}
	}

	// Build the new installed skills record
	newInstalledRecord := make(map[string]models.LockfileSkill)

	// Preserve previously installed skills that aren't in the current filter
	if len(options.SkillFilter) > 0 && options.OldSkillsState != nil {
		skillFilterMap := make(map[string]bool)
		for _, sf := range options.SkillFilter {
			skillFilterMap[sf] = true
		}
		for oldSkillName, oldSkillInfo := range options.OldSkillsState {
			if !skillFilterMap[oldSkillName] {
				newInstalledRecord[oldSkillName] = oldSkillInfo
			}
		}
	}

	for _, r := range installed {
		var skillTargets []string
		for _, d := range r.Destinations {
			skillTargets = append(skillTargets, d.Target)
		}
		sort.Strings(skillTargets)

		defaultTargetsSorted := make([]string, len(options.DefaultTargets))
		copy(defaultTargetsSorted, options.DefaultTargets)
		sort.Strings(defaultTargetsSorted)

		isOverride := strings.Join(skillTargets, ",") != strings.Join(defaultTargetsSorted, ",")

		skillVersion := ""
		if se, ok := manifest.Skills[r.SkillName]; ok {
			skillVersion = se.Version
		}

		ls := models.LockfileSkill{}
		if isOverride {
			ls.Targets = skillTargets
		}
		if skillVersion != "" {
			ls.Version = skillVersion
		}

		newInstalledRecord[r.SkillName] = ls
	}



	// Clean up Excluded list (remove newly installed skills from it)
	var newExcluded []string
	if options.OldExcluded != nil {
		installedMap := make(map[string]bool)
		for _, r := range installed {
			installedMap[r.SkillName] = true
		}
		for _, exc := range options.OldExcluded {
			if !installedMap[exc] {
				newExcluded = append(newExcluded, exc)
			}
		}
	}

	nsSource := source
	if fetchResult.Type == sources.SourceTypeLocal {
		nsSource = fetchResult.LocalPath
	}

	var targets []string
	if len(options.DefaultTargets) > 0 {
		targets = options.DefaultTargets
	}

	entry := models.NamespaceEntry{
		Source:    nsSource,
		Type:      string(fetchResult.Type),
		Ref:       fetchResult.Ref,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		SyncAll:   options.SyncAll,
		Excluded:  newExcluded,
		Targets:   targets,
		Skills: models.NamespaceSkills{
			Installed: newInstalledRecord,
		},
	}

	return UpsertNamespace(lockfile, namespaceName, entry)
}
