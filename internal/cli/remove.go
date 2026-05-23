package cli

import (
	"axen/internal/core"
	"axen/internal/ui"
	"axen/internal/utils"
	"context"

	"github.com/spf13/cobra"
)

func NewCmdRemove(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [namespace]",
		Short: "Remove a repository or specific skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			exclude, _ := cmd.Flags().GetBool("exclude")
			skillsStr, _ := cmd.Flags().GetStringSlice("skills")

			namespaceName := ""
			if len(args) > 0 {
				namespaceName = args[0]
			}

			return runRemove(cmd.Context(), deps, namespaceName, all, dryRun, exclude, skillsStr)
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Remove the entire repository and all its skills")
	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().BoolP("exclude", "e", false, "Automatically add removed skills to the exclude list (bypasses prompt)")
	cmd.Flags().StringSliceP("skills", "s", nil, "Comma-separated list of specific skills to remove")

	return cmd
}

func runRemove(ctx context.Context, deps *Dependencies, namespaceName string, all, dryRun, exclude bool, skillsFilter []string) error {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	if dryRun {
		ui.PrintDryRunBanner()
	}

	if namespaceName == "" {
		ns, err := ui.PromptNamespaceSelection(deps.Prompter, lockfile, false)
		if err != nil {
			return err
		}
		namespaceName = ns
	}

	nsEntry, exists := lockfile.Namespaces[namespaceName]
	if !exists {
		return utils.NewAxenError("namespace not found", "INVALID_NAMESPACE")
	}

	var skillsToRemove []string
	if !all && len(skillsFilter) == 0 {
		var installed []string
		for s := range nsEntry.Skills.Installed {
			installed = append(installed, s)
		}
		isAll, selected, err := ui.PromptRemoveMode(deps.Prompter, namespaceName, installed)
		if err != nil {
			return err
		}
		if isAll {
			all = true
			skillsToRemove = installed
		} else {
			if len(selected) == 0 {
				utils.Warn("No skills selected. Aborting.")
				return nil
			}
			skillsToRemove = selected
		}
	} else if all {
		for s := range nsEntry.Skills.Installed {
			skillsToRemove = append(skillsToRemove, s)
		}
	} else {
		skillsToRemove = skillsFilter
	}

	if nsEntry.SyncAll && !all && len(skillsToRemove) > 0 && !dryRun && !exclude {
		excludeMode, err := ui.PromptSyncAllRemoval(deps.Prompter, namespaceName)
		if err != nil {
			return err
		}
		if excludeMode {
			exclude = true
		} else {
			nsEntry.SyncAll = false
		}
	}

	if exclude && !dryRun {
		for _, s := range skillsToRemove {
			core.ExcludeSkillFromNamespace(lockfile, namespaceName, s)
		}
		nsEntry = lockfile.Namespaces[namespaceName]
	}

	for _, skillName := range skillsToRemove {
		skillInfo, ok := nsEntry.Skills.Installed[skillName]
		if !ok {
			continue
		}
		targetsToRemove := skillInfo.Targets
		if len(targetsToRemove) == 0 {
			targetsToRemove = nsEntry.Targets
		}
		removedFrom, err := core.UninstallSkillFromTargets(skillName, targetsToRemove, dryRun)
		if err != nil {
			return err
		}
		ui.PrintRemoveResults(skillName, removedFrom)

		if !dryRun {
			lockfile = core.RemoveSkillFromLockfile(lockfile, namespaceName, skillName)
		}
	}

	if !dryRun {
		_ = core.WriteLockfile(lockfile)
		
		if ns, ok := lockfile.Namespaces[namespaceName]; ok && len(ns.Skills.Installed) == 0 {
			removeSource, err := ui.PromptSourceRemoval(deps.Prompter, namespaceName)
			if err != nil {
				return err
			}
			if removeSource {
				lockfile = core.RemoveNamespace(lockfile, namespaceName)
				_ = core.WriteLockfile(lockfile)
				
				cache, _ := core.ReadSourcesIndex()
				delete(cache.Namespaces, namespaceName)
				_ = core.WriteSourcesIndex(cache)
				
				utils.Success("Removed source repository %s", namespaceName)
			}
		}
	}

	return nil
}
