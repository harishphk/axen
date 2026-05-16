package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Destination struct {
	Target string
	Path   string
}

type InstallResult struct {
	SkillName           string
	Destinations        []Destination
	SkippedDestinations []Destination
	Status              string // "installed" | "skipped" | "conflict" | "untracked_conflict"
}

type ConflictCandidate struct {
	Namespace   string
	IsInstalled bool
	IsExcluded  bool
}

type UntrackedConflict struct {
	SkillName string
	Target    string
	Path      string
}

type InstallOptions struct {
	Force                     bool
	DryRun                    bool
	Targets                   []string
	SkillFilter               []string // Filter skills to install
	ConflictStrategy          string   // "prompt", "keep", "overwrite"
	ConflictResolver          func(skillName string, candidates []ConflictCandidate) (string, error)
	UntrackedConflictResolver func(conflicts []UntrackedConflict) (map[string]bool, error)
}

func ExcludeSkillFromNamespace(lockfile *models.Lockfile, ns string, skillName string) {
	entry, ok := lockfile.Namespaces[ns]
	if !ok {
		return
	}
	delete(entry.Skills.Installed, skillName)
	isExcluded := false
	for _, ex := range entry.Excluded {
		if ex == skillName {
			isExcluded = true
			break
		}
	}
	if !isExcluded {
		entry.Excluded = append(entry.Excluded, skillName)
	}
	lockfile.Namespaces[ns] = entry
}

func isTargetTracked(lockfile *models.Lockfile, skillName string, target string) bool {
	if lockfile == nil {
		return false
	}
	for _, nsEntry := range lockfile.Namespaces {
		if ls, ok := nsEntry.Skills.Installed[skillName]; ok {
			ts := ls.Targets
			if len(ts) == 0 {
				ts = nsEntry.Targets
			}
			for _, t := range ts {
				if t == target {
					return true
				}
			}
		}
	}
	return false
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
		// Only install to targets where the agent tool is actually present on disk
		activeTargets = resolvers.GetDetectedTargets()
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
			_ = utils.RemoveDir(stagingDir)
		}
	}()

	if !options.DryRun {
		_ = utils.EnsureDir(stagingDir)
	}

	var stageWg sync.WaitGroup
	stageErrs := make(chan error, len(manifest.Skills)*len(allTargets)+1)

	var skillNames []string
	filterMap := make(map[string]bool)
	for _, f := range options.SkillFilter {
		filterMap[f] = true
	}

	for name := range manifest.Skills {
		if len(options.SkillFilter) > 0 && !filterMap[name] {
			continue
		}
		skillNames = append(skillNames, name)
	}
	sort.Strings(skillNames)

	var untrackedConflicts []UntrackedConflict
	for _, skillName := range skillNames {
		skillEntry, ok := manifest.Skills[skillName]
		if !ok {
			continue
		}

		cleanPath := filepath.Clean(skillEntry.Path)
		if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			continue
		}
		skillSourceDir := filepath.Join(sourceDir, cleanPath)
		if !utils.PathExists(skillSourceDir) {
			continue
		}

		hasSkillOverride := len(skillEntry.Targets) > 0
		skillTargets := manifest.Targets
		if hasSkillOverride {
			skillTargets = skillEntry.Targets
		} else if len(skillTargets) == 0 {
			skillTargets = activeTargets
		}

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
				continue
			}

			baseTargetDir := filepath.Clean(*targetPath)
			destPath := skillEntry.PathOverride
			if destPath == "" {
				destPath = filepath.Join(baseTargetDir, skillName)
			} else {
				cleanOverride := filepath.Clean(destPath)
				if strings.HasPrefix(cleanOverride, "..") || filepath.IsAbs(cleanOverride) {
					continue
				}
				destPath = filepath.Join(baseTargetDir, cleanOverride)
			}

			destPath = filepath.Clean(destPath)
			if !strings.HasPrefix(destPath, baseTargetDir+string(filepath.Separator)) && destPath != baseTargetDir {
				continue
			}

			if utils.PathExists(destPath) && !isTargetTracked(lockfile, skillName, target) {
				untrackedConflicts = append(untrackedConflicts, UntrackedConflict{
					SkillName: skillName,
					Target:    target,
					Path:      destPath,
				})
			}
		}
	}

	skipTargets := make(map[string]bool)
	if len(untrackedConflicts) > 0 {
		if options.Force {
			// Overwrite all, so skipTargets remains empty
		} else if options.UntrackedConflictResolver != nil {
			decisions, err := options.UntrackedConflictResolver(untrackedConflicts)
			if err != nil {
				return nil, err
			}
			for _, c := range untrackedConflicts {
				key := c.SkillName + ":" + c.Target
				if !decisions[key] {
					skipTargets[key] = true
				}
			}
		} else {
			// Non-interactive: skip all
			for _, c := range untrackedConflicts {
				key := c.SkillName + ":" + c.Target
				skipTargets[key] = true
			}
		}
	}


	for _, skillName := range skillNames {
		skillEntry := manifest.Skills[skillName]

		cleanPath := filepath.Clean(skillEntry.Path)
		if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			utils.Warn("Skill %q has an invalid path %q, skipping", skillName, skillEntry.Path)
			results = append(results, InstallResult{SkillName: skillName, Status: "skipped"})
			continue
		}
		skillSourceDir := filepath.Join(sourceDir, cleanPath)

		if !utils.PathExists(skillSourceDir) {
			utils.Warn("Skill %q not found at %s, skipping", skillName, skillEntry.Path)
			results = append(results, InstallResult{SkillName: skillName, Status: "skipped"})
			continue
		}

		existingOwnerNs, _ := FindSkillNamespace(lockfile, skillName)
		if existingOwnerNs != "" && existingOwnerNs != namespaceName {
			if options.Force || options.ConflictStrategy == "overwrite" {
				ExcludeSkillFromNamespace(lockfile, existingOwnerNs, skillName)
			} else if !options.DryRun && (options.ConflictStrategy == "prompt" || options.ConflictStrategy == "") {
				cache, _ := ReadSourcesIndex()

				var candidates []ConflictCandidate
				candidates = append(candidates, ConflictCandidate{Namespace: existingOwnerNs, IsInstalled: true, IsExcluded: false})
				candidates = append(candidates, ConflictCandidate{Namespace: namespaceName, IsInstalled: false, IsExcluded: false})

				if cache != nil {
					for cNs, cData := range cache.Namespaces {
						if cNs == existingOwnerNs || cNs == namespaceName {
							continue
						}
						if _, ok := cData.Available[skillName]; ok {
							isExcluded := false
							if lockfileNs, ok := lockfile.Namespaces[cNs]; ok {
								for _, ex := range lockfileNs.Excluded {
									if ex == skillName {
										isExcluded = true
										break
									}
								}
							}
							if !isExcluded {
								candidates = append(candidates, ConflictCandidate{Namespace: cNs, IsInstalled: false, IsExcluded: false})
							}
						}
					}
				}

				if options.ConflictResolver != nil {
					selectedNs, err := options.ConflictResolver(skillName, candidates)
					if err != nil {
						return nil, err
					}

					if selectedNs != namespaceName {
						if selectedNs != existingOwnerNs {
							utils.Info("To activate %q from %q, run: axen install %s --skills %s", skillName, selectedNs, selectedNs, skillName)
						}
						results = append(results, InstallResult{SkillName: skillName, Status: "conflict"})
						continue
					} else {
						ExcludeSkillFromNamespace(lockfile, existingOwnerNs, skillName)
					}
				} else {
					utils.Warn(`⚠ Conflict: "%s" already installed from "%s".`, skillName, existingOwnerNs)
					results = append(results, InstallResult{SkillName: skillName, Status: "conflict"})
					continue
				}
			} else {
				utils.Warn(`⚠ Conflict: "%s" already installed from "%s".`, skillName, existingOwnerNs)
				results = append(results, InstallResult{SkillName: skillName, Status: "conflict"})
				continue
			}
		}

		hasSkillOverride := len(skillEntry.Targets) > 0
		skillTargets := manifest.Targets
		if hasSkillOverride {
			skillTargets = skillEntry.Targets
		} else if len(skillTargets) == 0 {
			skillTargets = activeTargets
		}

		var destinations []Destination
		var skippedDestinations []Destination

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

			baseTargetDir := filepath.Clean(*targetPath)
			destPath := skillEntry.PathOverride
			if destPath == "" {
				destPath = filepath.Join(baseTargetDir, skillName)
			} else {
				cleanOverride := filepath.Clean(destPath)
				if strings.HasPrefix(cleanOverride, "..") || filepath.IsAbs(cleanOverride) {
					utils.Warn("Skill %q has an invalid path_override %q, skipping target %s", skillName, destPath, target)
					continue
				}
				destPath = filepath.Join(baseTargetDir, cleanOverride)
			}

			destPath = filepath.Clean(destPath)
			if !strings.HasPrefix(destPath, baseTargetDir+string(filepath.Separator)) && destPath != baseTargetDir {
				utils.Warn("Skill %q destination %q escapes target directory %q, skipping", skillName, destPath, baseTargetDir)
				continue
			}

			key := skillName + ":" + target
			if skipTargets[key] {
				skippedDestinations = append(skippedDestinations, Destination{Target: target, Path: destPath})
				continue
			}

			destinations = append(destinations, Destination{Target: target, Path: destPath})

			if options.DryRun {
				utils.Info("  INSTALL  %s → %s (%s)", skillName, destPath, target)
			} else {
				stagingSkillDir := filepath.Join(stagingDir, target, skillName)
				stageWg.Add(1)
				go func(src, dest string) {
					defer stageWg.Done()
					if err := utils.CopyDir(src, dest); err != nil {
						stageErrs <- err
					}
				}(skillSourceDir, stagingSkillDir)
			}
		}

		status := "installed"
		if len(destinations) == 0 {
			if len(skippedDestinations) > 0 {
				status = "untracked_conflict"
			} else {
				status = "skipped"
			}
		}
		results = append(results, InstallResult{
			SkillName:           skillName,
			Destinations:        destinations,
			SkippedDestinations: skippedDestinations,
			Status:              status,
		})
	}

	if !options.DryRun {
		stageWg.Wait()
		close(stageErrs)
		for err := range stageErrs {
			return nil, fmt.Errorf("failed to copy to staging: %w", err)
		}

		type rollbackInfo struct {
			Dest       Destination
			BackupPath string
		}

		var mu sync.Mutex
		var successfulCopies []Destination
		var backups []rollbackInfo

		var finalWg sync.WaitGroup
		finalErrs := make(chan error, len(results)*len(allTargets)+1)

		for _, result := range results {
			if result.Status != "installed" {
				continue
			}
			for _, dest := range result.Destinations {
				stagingSkillDir := filepath.Join(stagingDir, dest.Target, result.SkillName)
				finalWg.Add(1)
				go func(src, dst string, d Destination, skillName string) {
					defer finalWg.Done()

					backupPath := filepath.Join(stagingDir, "backup", d.Target, skillName)
					if utils.PathExists(dst) {
						if err := utils.CopyDir(dst, backupPath); err != nil {
							finalErrs <- fmt.Errorf("failed to back up %s: %w", dst, err)
							return
						}
						mu.Lock()
						backups = append(backups, rollbackInfo{Dest: d, BackupPath: backupPath})
						mu.Unlock()

						if err := utils.RemoveDir(dst); err != nil {
							finalErrs <- fmt.Errorf("failed to remove original %s: %w", dst, err)
							return
						}
					}

					if err := utils.EnsureDir(filepath.Dir(dst)); err != nil {
						finalErrs <- err
						return
					}
					if err := utils.CopyDir(src, dst); err != nil {
						finalErrs <- err
						return
					}

					mu.Lock()
					successfulCopies = append(successfulCopies, d)
					mu.Unlock()
				}(stagingSkillDir, dest.Path, dest, result.SkillName)
			}
		}

		finalWg.Wait()
		close(finalErrs)

		var firstErr error
		for err := range finalErrs {
			firstErr = err
			break
		}

		if firstErr != nil {
			utils.Warn("Installation failed. Rolling back changes...")

			// 1. Delete newly installed folders
			for _, d := range successfulCopies {
				if utils.PathExists(d.Path) {
					_ = utils.RemoveDir(d.Path)
				}
			}

			// 2. Restore backed up folders
			for _, b := range backups {
				if utils.PathExists(b.Dest.Path) {
					_ = utils.RemoveDir(b.Dest.Path)
				}
				if err := utils.CopyDir(b.BackupPath, b.Dest.Path); err != nil {
					utils.Warn("Rollback failed to restore backup at %s: %v", b.Dest.Path, err)
				}
			}

			return nil, fmt.Errorf("failed to install skill: %w", firstErr)
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
				_ = utils.RemoveDir(skillPath)
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
