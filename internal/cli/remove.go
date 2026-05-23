package cli

import (
	"axen/internal/core"
	"axen/internal/ui"
	"axen/internal/utils"
	"context"

	"github.com/pterm/pterm"
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

			opts := RunRemoveOptions{
				All:          all,
				DryRun:       dryRun,
				Exclude:      exclude,
				SkillsFilter: skillsStr,
			}

			return runRemove(cmd.Context(), deps, namespaceName, opts)
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Remove the entire repository and all its skills")
	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().BoolP("exclude", "e", false, "Automatically add removed skills to the exclude list (bypasses prompt)")
	cmd.Flags().StringSliceP("skills", "s", nil, "Comma-separated list of specific skills to remove")

	return cmd
}

type RunRemoveOptions struct {
	All            bool
	DryRun         bool
	Exclude        bool
	SkillsFilter   []string
	IsSourceRemove bool
}

func runRemove(ctx context.Context, deps *Dependencies, namespaceName string, opts RunRemoveOptions) error {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	if opts.DryRun {
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

	if opts.IsSourceRemove && !opts.DryRun {
		confirm, err := deps.Prompter.InteractiveConfirm("Are you sure you want to completely remove the source repository '"+namespaceName+"' and uninstall all its skills?", *pterm.DefaultInteractiveConfirm.WithDefaultValue(true))
		if err != nil {
			return err
		}
		if !confirm {
			utils.Warn("Source removal aborted.")
			return nil
		}
	}

	var skillsToRemove []string
	if !opts.All && len(opts.SkillsFilter) == 0 {
		var installed []string
		for s := range nsEntry.Skills.Installed {
			installed = append(installed, s)
		}
		isAll, selected, err := ui.PromptRemoveMode(deps.Prompter, namespaceName, installed)
		if err != nil {
			return err
		}
		if isAll {
			opts.All = true
			skillsToRemove = installed
		} else {
			if len(selected) == 0 {
				utils.Warn("No skills selected. Aborting.")
				return nil
			}
			skillsToRemove = selected
		}
	} else if opts.All {
		for s := range nsEntry.Skills.Installed {
			skillsToRemove = append(skillsToRemove, s)
		}
	} else {
		skillsToRemove = opts.SkillsFilter
	}

	if nsEntry.SyncAll && !opts.All && len(skillsToRemove) > 0 && !opts.DryRun && !opts.Exclude {
		excludeMode, err := ui.PromptSyncAllRemoval(deps.Prompter, namespaceName)
		if err != nil {
			return err
		}
		if excludeMode {
			opts.Exclude = true
		} else {
			nsEntry.SyncAll = false
		}
	}

	if opts.Exclude && !opts.DryRun {
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
		removedFrom, err := core.UninstallSkillFromTargets(skillName, targetsToRemove, opts.DryRun)
		if err != nil {
			return err
		}
		ui.PrintRemoveResults(skillName, removedFrom)

		if !opts.DryRun {
			lockfile = core.RemoveSkillFromLockfile(lockfile, namespaceName, skillName)
		}
	}

	if !opts.DryRun {
		_ = core.WriteLockfile(lockfile)

		if ns, ok := lockfile.Namespaces[namespaceName]; ok && len(ns.Skills.Installed) == 0 {
			removeSource := true
			if !opts.IsSourceRemove {
				var err error
				removeSource, err = ui.PromptSourceRemoval(deps.Prompter, namespaceName)
				if err != nil {
					return err
				}
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
