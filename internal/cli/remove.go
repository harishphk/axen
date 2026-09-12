package cli

import (
	"context"
	"sort"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/ui"
	"github.com/harishphk/axen/internal/utils"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type RunRemoveOptions struct {
	All            bool
	DryRun         bool
	Exclude        bool
	SkillsFilter   []string
	BundleFilter   []string
	IsSourceRemove bool
}

func NewCmdRemove(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [namespace]",
		Short: "Remove a repository or specific skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			all, _ := cmd.Flags().GetBool("all")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			exclude, _ := cmd.Flags().GetBool("exclude")
			skillsStr, _ := cmd.Flags().GetStringSlice("skills")
			bundleStr, _ := cmd.Flags().GetStringSlice("bundle")

			namespaceName := ""
			if len(args) > 0 {
				namespaceName = args[0]
			}

			opts := RunRemoveOptions{
				All:          all,
				DryRun:       dryRun,
				Exclude:      exclude,
				SkillsFilter: skillsStr,
				BundleFilter: bundleStr,
			}

			return runRemove(cmd.Context(), deps, namespaceName, opts)
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Remove the entire repository and all its skills")
	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().BoolP("exclude", "e", false, "Automatically add removed skills to the exclude list (bypasses prompt)")
	cmd.Flags().StringSliceP("skills", "s", nil, "Comma-separated list of specific skills to remove")
	cmd.Flags().StringSliceP("bundle", "b", nil, "Comma-separated list of specific bundles to remove")

	cmd.MarkFlagsMutuallyExclusive("all", "skills")
	cmd.MarkFlagsMutuallyExclusive("all", "bundle")

	return cmd
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
	var bundlesToRemove []string

	if !opts.All && len(opts.SkillsFilter) == 0 && len(opts.BundleFilter) == 0 {
		var installed []string
		for s := range nsEntry.Skills.Installed {
			installed = append(installed, s)
		}
		sort.Strings(installed)
		isAll, selectedSkills, selectedBundles, err := ui.PromptRemoveMode(deps.Prompter, namespaceName, installed, nsEntry.Bundles)
		if err != nil {
			return err
		}
		if isAll {
			opts.All = true
			skillsToRemove = installed
			bundlesToRemove = nsEntry.Bundles
		} else {
			if len(selectedSkills) == 0 && len(selectedBundles) == 0 {
				utils.Warn("No skills or bundles selected. Aborting.")
				return nil
			}
			skillsToRemove = selectedSkills
			bundlesToRemove = selectedBundles
		}
	} else if opts.All {
		for s := range nsEntry.Skills.Installed {
			skillsToRemove = append(skillsToRemove, s)
		}
		sort.Strings(skillsToRemove)
		bundlesToRemove = nsEntry.Bundles
	} else {
		skillsToRemove = opts.SkillsFilter
		bundlesToRemove = opts.BundleFilter
	}

	spec := core.RemoveSpec{
		NamespaceName: namespaceName,
		RemoveAll:     opts.All,
		RemoveSkills:  skillsToRemove,
		RemoveBundles: bundlesToRemove,
		Exclude:       opts.Exclude,
		PromptExclude: func(ns string) (bool, error) {
			return ui.PromptSyncAllRemoval(deps.Prompter, ns)
		},
		OnNamespaceEmpty: func(ns string) (bool, error) {
			if opts.IsSourceRemove || opts.All {
				return true, nil
			}
			return ui.PromptSourceRemoval(deps.Prompter, ns)
		},
		OnRemovedSource: func(ns string) {
			utils.Success("Removed source repository %s", pterm.Cyan(ns))
		},
		DryRun: opts.DryRun,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
	}

	result, err := deps.Engine.Remove(ctx, spec)
	if err != nil {
		return err
	}

	pterm.Println()
	for _, p := range result.Pruned {
		ui.PrintRemoveResults(p.SkillName, p.Removed)
	}

	if !opts.DryRun && opts.IsSourceRemove {
		utils.Success("Successfully removed source %s!", pterm.Cyan(namespaceName))
	}

	return nil
}
