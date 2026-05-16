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

	return cmd
}

type RunRemoveOptions struct {
	All            bool
	DryRun         bool
	Exclude        bool
	SkillsFilter   []string
	BundleFilter   []string
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

	// --- Determine what to remove ---
	var skillsToRemove []string
	var bundlesToRemove []string

	if !opts.All && len(opts.SkillsFilter) == 0 && len(opts.BundleFilter) == 0 {
		var installed []string
		for s := range nsEntry.Skills.Installed {
			installed = append(installed, s)
		}
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
		bundlesToRemove = nsEntry.Bundles
	} else {
		skillsToRemove = opts.SkillsFilter
		bundlesToRemove = opts.BundleFilter
	}

	// --- Build IntentAction ---
	action := core.IntentAction{
		RemoveBundles: bundlesToRemove,
		RemoveSkills:  skillsToRemove,
	}

	// Fetch manifest early — needed for bundle resolution in SyncAll handling
	spinner, _ := utils.StartSpinner("Computing state for " + namespaceName + "...")
	fetchResult, manifest, err := core.FetchAndResolve(ctx, nsEntry.Source, namespaceName)
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	spinner.Success("Computed state")

	// PRE-HOOK: SyncAll handling
	// When SyncAll is true and we're removing skills or bundles, we need to
	// either exclude or disable SyncAll. Otherwise removal has no effect since
	// SyncAll overrides by making ALL manifest skills desired.
	currentIntent := core.GetIntent(lockfile, namespaceName)
	isPartialRemove := !opts.All && (len(skillsToRemove) > 0 || len(bundlesToRemove) > 0)
	if currentIntent.SyncAll && isPartialRemove && !opts.DryRun && !opts.Exclude {
		if len(skillsToRemove) > 0 {
			excludeMode, err := ui.PromptSyncAllRemoval(deps.Prompter, namespaceName)
			if err != nil {
				return err
			}
			if excludeMode {
				opts.Exclude = true
			}
		}

		if !opts.Exclude {
			// Disable SyncAll and convert all currently installed skills
			// (except the ones being removed) to explicit skills.
			// Skills that belong to remaining bundles stay as bundle-tracked.
			syncAllFalse := false
			action.SetSyncAll = &syncAllFalse

			// Compute which bundles remain after removal
			remainingBundles := make(map[string]bool)
			for _, b := range currentIntent.Bundles {
				remainingBundles[b] = true
			}
			for _, b := range bundlesToRemove {
				delete(remainingBundles, b)
			}

			// Figure out which skills are covered by remaining bundles
			bundleCoveredSkills := make(map[string]bool)
			for bName := range remainingBundles {
				if bundle, ok := manifest.Bundles[bName]; ok {
					for _, s := range bundle.Skills {
						bundleCoveredSkills[s] = true
					}
				}
			}

			// Skills not covered by remaining bundles need to be explicit
			removeSet := make(map[string]bool)
			for _, r := range skillsToRemove {
				removeSet[r] = true
			}
			// Also compute bundle skills being removed
			for _, bName := range bundlesToRemove {
				if bundle, ok := manifest.Bundles[bName]; ok {
					for _, s := range bundle.Skills {
						removeSet[s] = true
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
				action.AddSkills = append(action.AddSkills, skillName)
			}
		}
	}

	if opts.All {
		syncAllFalse := false
		action.SetSyncAll = &syncAllFalse
	}

	// Exclude flag: add removed skills to the excluded list
	if opts.Exclude && len(skillsToRemove) > 0 {
		action.AddExcluded = skillsToRemove
	}

	// Merge intent
	newIntent := core.MergeIntent(currentIntent, action)

	// Reconcile
	result, err := core.Reconcile(ctx, lockfile, namespaceName, nsEntry.Source, fetchResult, manifest, newIntent, core.ReconcileOpts{
		DryRun: opts.DryRun,
	})
	if err != nil {
		return err
	}

	// Print results
	for _, p := range result.Pruned {
		ui.PrintRemoveResults(p.SkillName, p.Removed)
	}

	// POST-HOOK: If no skills remain, prompt to remove the source
	if !opts.DryRun {
		if ns, ok := lockfile.Namespaces[namespaceName]; ok && len(ns.Skills.Installed) == 0 {
			removeSource := true
			if !opts.IsSourceRemove {
				var promptErr error
				removeSource, promptErr = ui.PromptSourceRemoval(deps.Prompter, namespaceName)
				if promptErr != nil {
					return promptErr
				}
			}
			if removeSource {
				lockfile = core.RemoveNamespace(lockfile, namespaceName)
				_ = core.WriteLockfile(lockfile)

				cache, _ := core.ReadSourcesIndex()
				if cache != nil {
					delete(cache.Namespaces, namespaceName)
					_ = core.WriteSourcesIndex(cache)
				}

				utils.Success("Removed source repository %s", namespaceName)
			}
		}
	}

	return nil
}
