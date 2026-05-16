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

	newSkillsRecord := make(map[string]models.LockfileSkill)

	if len(options.SkillFilter) > 0 && options.OldSkillsState != nil {
		skillFilterMap := make(map[string]bool)
		for _, sf := range options.SkillFilter {
			skillFilterMap[sf] = true
		}
		for oldSkillName, oldSkillInfo := range options.OldSkillsState {
			if !skillFilterMap[oldSkillName] {
				newSkillsRecord[oldSkillName] = oldSkillInfo
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

		newSkillsRecord[r.SkillName] = ls
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
		Targets:   targets,
		Skills:    newSkillsRecord,
	}

	return UpsertNamespace(lockfile, namespaceName, entry)
}
