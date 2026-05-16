package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"fmt"
	"path/filepath"
	"strings"
)

type Destination struct {
	Target string
	Path   string
}

type InstallResult struct {
	SkillName    string
	Destinations []Destination
	Status       string // "installed" | "skipped" | "conflict"
}

type InstallOptions struct {
	Force   bool
	DryRun  bool
	Targets []string
}

func InstallSkills(
	sourceDir string,
	manifest *models.Manifest,
	lockfile *models.Lockfile,
	namespaceName string,
	options InstallOptions,
) ([]InstallResult, error) {
	var results []InstallResult
	stagingDir := resolvers.GetStagingDir()

	allTargets := resolvers.GetKnownTargets()
	activeTargets := options.Targets
	if len(activeTargets) == 0 {
		activeTargets = allTargets
	}

	if len(options.Targets) > 0 {
		for _, t := range options.Targets {
			if !resolvers.IsKnownTarget(t) {
				return nil, utils.NewAxenError(fmt.Sprintf(`Unknown target: "%s". Known targets: %s`, t, strings.Join(allTargets, ", ")), "INVALID_TARGET")
			}
		}
	}

	defer func() {
		if !options.DryRun && utils.PathExists(stagingDir) {
			utils.RemoveDir(stagingDir)
		}
	}()

	if !options.DryRun {
		utils.EnsureDir(stagingDir)
	}

	for skillName, skillEntry := range manifest.Skills {
		skillSourceDir := filepath.Join(sourceDir, skillEntry.Path)

		if !utils.PathExists(skillSourceDir) {
			utils.Warn("Skill %q not found at %s, skipping", skillName, skillEntry.Path)
			results = append(results, InstallResult{SkillName: skillName, Status: "skipped"})
			continue
		}

		existingOwnerNs, _ := FindSkillNamespace(lockfile, skillName)
		if existingOwnerNs != "" && existingOwnerNs != namespaceName && !options.Force {
			utils.Warn(`⚠ Conflict: "%s" already installed from "%s". Use --force to overwrite.`, skillName, existingOwnerNs)
			results = append(results, InstallResult{SkillName: skillName, Status: "conflict"})
			continue
		}

		hasSkillOverride := len(skillEntry.Targets) > 0
		skillTargets := manifest.Targets
		if hasSkillOverride {
			skillTargets = skillEntry.Targets
		} else if len(skillTargets) == 0 {
			skillTargets = activeTargets
		}

		var destinations []Destination

		for _, target := range skillTargets {
			isActive := false
			for _, at := range activeTargets {
				if at == target {
					isActive = true
					break
				}
			}

			if !hasSkillOverride && !isActive {
				continue
			}

			targetPath := resolvers.ResolveTargetPath(target)
			if targetPath == nil {
				utils.Warn("Unknown target %q, skipping", target)
				continue
			}

			destPath := skillEntry.PathOverride
			if destPath == "" {
				destPath = filepath.Join(*targetPath, skillName)
			}
			destinations = append(destinations, Destination{Target: target, Path: destPath})

			if options.DryRun {
				utils.Info("  INSTALL  %s → %s (%s)", skillName, destPath, target)
			} else {
				stagingSkillDir := filepath.Join(stagingDir, target, skillName)
				utils.CopyDir(skillSourceDir, stagingSkillDir)
			}
		}

		results = append(results, InstallResult{SkillName: skillName, Destinations: destinations, Status: "installed"})
	}

	if !options.DryRun {
		for _, result := range results {
			if result.Status != "installed" {
				continue
			}
			for _, dest := range result.Destinations {
				stagingSkillDir := filepath.Join(stagingDir, dest.Target, result.SkillName)
				utils.EnsureDir(filepath.Dir(dest.Path))
				if utils.PathExists(dest.Path) {
					utils.RemoveDir(dest.Path)
				}
				utils.CopyDir(stagingSkillDir, dest.Path)
			}
		}
	}

	return results, nil
}

func UninstallSkillFromTargets(skillName string, targets []string, dryRun bool) ([]Destination, error) {
	if skillName == "" || skillName == "." || skillName == ".." || strings.Contains(skillName, "/") || strings.Contains(skillName, "\\") {
		return nil, fmt.Errorf("critical safety error: Invalid skill name %q prevents destructive operations", skillName)
	}

	var removed []Destination
	for _, target := range targets {
		targetPath := resolvers.ResolveTargetPath(target)
		if targetPath == nil {
			continue
		}
		skillPath := filepath.Join(*targetPath, skillName)
		if utils.PathExists(skillPath) {
			if dryRun {
				utils.Info("  REMOVE  %s ← %s (%s)", skillName, skillPath, target)
			} else {
				utils.RemoveDir(skillPath)
			}
			removed = append(removed, Destination{Target: target, Path: skillPath})
		}
	}
	return removed, nil
}

type PrunedSkill struct {
	SkillName string
	Removed   []Destination
}

func PruneSkills(
	oldSkillsState map[string]models.LockfileSkill,
	newInstallResults []InstallResult,
	oldNamespaceTargets []string,
	dryRun bool,
	filter []string,
) ([]PrunedSkill, error) {
	var prunes []PrunedSkill

	newState := make(map[string][]string)
	for _, res := range newInstallResults {
		if res.Status == "installed" {
			var tgs []string
			for _, d := range res.Destinations {
				tgs = append(tgs, d.Target)
			}
			newState[res.SkillName] = tgs
		}
	}

	filterMap := make(map[string]bool)
	for _, f := range filter {
		filterMap[f] = true
	}

	for skillName, oldInfo := range oldSkillsState {
		if len(filter) > 0 && !filterMap[skillName] {
			continue
		}

		newTargets, exists := newState[skillName]
		oldTargets := oldInfo.Targets
		if len(oldTargets) == 0 {
			oldTargets = oldNamespaceTargets
		}

		if !exists {
			removed, _ := UninstallSkillFromTargets(skillName, oldTargets, dryRun)
			if len(removed) > 0 {
				prunes = append(prunes, PrunedSkill{SkillName: skillName, Removed: removed})
			}
		} else {
			var targetsToRemove []string
			newTargetsMap := make(map[string]bool)
			for _, nt := range newTargets {
				newTargetsMap[nt] = true
			}
			for _, ot := range oldTargets {
				if !newTargetsMap[ot] {
					targetsToRemove = append(targetsToRemove, ot)
				}
			}

			if len(targetsToRemove) > 0 {
				removed, _ := UninstallSkillFromTargets(skillName, targetsToRemove, dryRun)
				if len(removed) > 0 {
					prunes = append(prunes, PrunedSkill{SkillName: skillName, Removed: removed})
				}
			}
		}
	}

	return prunes, nil
}
