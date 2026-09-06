package cli

import (
	"context"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/services"
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

	installSvc := &services.InstallService{}
	
	var spinner *pterm.SpinnerPrinter
	fetchOpts := services.FetchOptions{
		OnFetchStart: func(ns string) {
			spinner, _ = utils.StartSpinner("Computing state for " + namespaceName + "...")
		},
		OnFetchDone: func(ns string, err error) {
			if err != nil {
				spinner.Fail("Failed to fetch state")
			}
		},
	}

	fetchResult, manifest, err := installSvc.FetchManifest(ctx, nsEntry.Source, namespaceName, fetchOpts)
	if err != nil {
		return err
	}
	if spinner != nil {
		spinner.Success("Computed state")
	}

	currentIntent := core.GetIntent(lockfile, namespaceName)
	isPartialRemove := !opts.All && (len(skillsToRemove) > 0 || len(bundlesToRemove) > 0)
	var setSyncAll *bool
	var addSkills []string
	var addExcluded []string

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
			for bName := range remainingBundles {
				if bundle, ok := manifest.Bundles[bName]; ok {
					for _, s := range bundle.Skills {
						bundleCoveredSkills[s] = true
					}
				}
			}

			removeSet := make(map[string]bool)
			for _, r := range skillsToRemove {
				removeSet[r] = true
			}
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
				addSkills = append(addSkills, skillName)
			}
		}
	}

	if opts.All {
		f := false
		setSyncAll = &f
	}

	if opts.Exclude && len(skillsToRemove) > 0 {
		addExcluded = skillsToRemove
	}

	req := services.RemoveRequest{
		NamespaceName: namespaceName,
		SourceURL:     nsEntry.Source,
		FetchResult:   fetchResult,
		Manifest:      manifest,
		RemoveSkills:  skillsToRemove,
		RemoveBundles: bundlesToRemove,
		AddSkills:     addSkills,
		AddExcluded:   addExcluded,
		SetSyncAll:    setSyncAll,
		RemoveAll:     opts.All,
		DryRun:        opts.DryRun,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
	}

	svc := &services.RemoveService{}
	result, err := svc.Remove(ctx, req)
	if err != nil {
		return err
	}

	pterm.Println()
	for _, p := range result.Pruned {
		ui.PrintRemoveResults(p.SkillName, p.Removed)
	}

	if !opts.DryRun {
		if opts.IsSourceRemove {
			utils.Success("Successfully removed source %s!", pterm.Cyan(namespaceName))
		}

		// Re-read lockfile to check if namespace is empty
		lockfile, _ = core.ReadLockfile()
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

				utils.Success("Removed source repository %s", pterm.Cyan(namespaceName))
			}
		}
	}

	return nil
}
