package cli

import (
	"axen/internal/core"
	"axen/internal/resolvers"
	"axen/internal/ui"
	"axen/internal/utils"
	"context"

	"github.com/spf13/cobra"
)

type RunInstallOptions struct {
	Force            bool
	DryRun           bool
	AllSkills        bool
	TargetFilter     []string
	SkillFilter      []string
	BundleFilter     []string
	ConflictStrategy string
	SkipAutoDetect   bool // skip default bundle auto-detection (used by source add)
}

func NewCmdInstall(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [namespace]",
		Short: "Install skills from a registered source or local directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			force, _ := cmd.Flags().GetBool("force")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			targetsStr, _ := cmd.Flags().GetStringSlice("targets")
			skillsStr, _ := cmd.Flags().GetStringSlice("skills")
			bundleStr, _ := cmd.Flags().GetStringSlice("bundle")
			allFlag, _ := cmd.Flags().GetBool("all")
			conflictStrategy, _ := cmd.Flags().GetString("conflict-strategy")

			opts := RunInstallOptions{
				Force:            force,
				DryRun:           dryRun,
				AllSkills:        allFlag,
				TargetFilter:     targetsStr,
				SkillFilter:      skillsStr,
				BundleFilter:     bundleStr,
				ConflictStrategy: conflictStrategy,
			}

			namespaceName := ""
			if len(args) > 0 {
				namespaceName = args[0]
			}

			return runInstall(cmd.Context(), deps, namespaceName, opts)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Overwrite existing skills on conflict")
	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().BoolP("all", "a", false, "Install all skills, bypassing interactive prompt")
	cmd.Flags().StringSliceP("targets", "t", nil, "Comma-separated list of targets to install into")
	cmd.Flags().StringSliceP("skills", "s", nil, "Comma-separated list of specific skills to install")
	cmd.Flags().StringSliceP("bundle", "b", nil, "Comma-separated list of bundles to install")
	cmd.Flags().StringP("conflict-strategy", "c", "prompt", "Conflict resolution strategy: prompt, overwrite, keep")

	return cmd
}

func runInstall(ctx context.Context, deps *Dependencies, namespaceName string, opts RunInstallOptions) error {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	if opts.DryRun {
		ui.PrintDryRunBanner()
	}

	// --- Resolve namespace ---
	if namespaceName == "" {
		scanned, _ := core.ScanSkills(".")
		hasLocal := core.HasManifest(".") || len(scanned) > 0
		ns, err := ui.PromptNamespaceSelection(deps.Prompter, lockfile, hasLocal)
		if err != nil {
			return err
		}
		namespaceName = ns
	}

	sourceURL := namespaceName
	if namespaceName != "." {
		nsEntry, exists := lockfile.Namespaces[namespaceName]
		if !exists {
			return utils.NewAxenError("source not found", "INVALID_SOURCE")
		}
		sourceURL = nsEntry.Source
	}

	// --- Fetch source ---
	spinner, _ := utils.StartSpinner("Fetching " + namespaceName + "...")
	fetchResult, manifest, err := core.FetchAndResolve(ctx, sourceURL, namespaceName)
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	spinner.Success("Fetched " + namespaceName)

	// --- Build IntentAction from flags / interactive prompts ---
	currentIntent := core.GetIntent(lockfile, namespaceName)
	action := core.IntentAction{
		AddBundles: opts.BundleFilter,
		AddSkills:  opts.SkillFilter,
	}

	// Validate bundle names
	for _, bName := range opts.BundleFilter {
		if _, ok := manifest.Bundles[bName]; !ok {
			utils.Warn("Bundle \"%s\" not found in manifest, skipping", bName)
		}
	}

	if opts.AllSkills {
		syncAll := true
		action.SetSyncAll = &syncAll
	}
	// Track what skills the user selected (from any path) for scoping
	var effectiveScope []string

	// If no explicit flags, try default bundle or interactive prompt
	if len(opts.BundleFilter) == 0 && len(opts.SkillFilter) == 0 && !opts.AllSkills {
		// Check for default bundle (only when not called from source add)
		defaultBundleFound := false
		if !opts.SkipAutoDetect {
			for name, b := range manifest.Bundles {
				if b.IsDefault {
					utils.Warn("Auto-installing default bundle: " + name)
					action.AddBundles = []string{name}
					effectiveScope = append(effectiveScope, b.Skills...)
					defaultBundleFound = true
					break
				}
			}
		}

		if !defaultBundleFound {
			// Interactive prompt
			var newSkillNames []string
			for s := range manifest.Skills {
				if entry, ok := lockfile.Namespaces[namespaceName]; !ok || func() bool { _, e := entry.Skills.Installed[s]; return !e }() {
					newSkillNames = append(newSkillNames, s)
				}
			}

			installedCount := 0
			if entry, ok := lockfile.Namespaces[namespaceName]; ok {
				installedCount = len(entry.Skills.Installed)
			}

			selectedSkills, selectedBundles, isAll, err := ui.PromptSkillSelection(deps.Prompter, namespaceName, newSkillNames, installedCount, len(manifest.Skills), manifest.Bundles)
			if err != nil {
				return err
			}
			if len(selectedSkills) == 0 && !isAll {
				utils.Warn("No skills selected. Aborting.")
				return nil
			}

			if isAll {
				syncAll := true
				action.SetSyncAll = &syncAll
			} else if len(selectedBundles) > 0 {
				action.AddBundles = selectedBundles
				for _, bName := range selectedBundles {
					if b, ok := manifest.Bundles[bName]; ok {
						effectiveScope = append(effectiveScope, b.Skills...)
					}
				}
			} else {
				action.AddSkills = selectedSkills
				effectiveScope = append(effectiveScope, selectedSkills...)
			}
		}
	} else {
		// Explicit flags: build scope from CLI args
		effectiveScope = append(effectiveScope, opts.SkillFilter...)
		for _, bName := range opts.BundleFilter {
			if b, ok := manifest.Bundles[bName]; ok {
				effectiveScope = append(effectiveScope, b.Skills...)
			}
		}
	}

	reconcileOpts := core.ReconcileOpts{
		Force:            opts.Force,
		DryRun:           opts.DryRun,
		ConflictStrategy: opts.ConflictStrategy,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
		UntrackedConflictResolver: func(conflicts []core.UntrackedConflict) (map[string]bool, error) {
			return ui.PromptUntrackedConflicts(deps.Prompter, conflicts)
		},
	}

	// --- Resolve targets ---
	var selectedTargets []string
	if len(opts.TargetFilter) > 0 {
		selectedTargets = opts.TargetFilter
	} else if !opts.AllSkills {
		var err error
		selectedTargets, err = ui.PromptTargetSelection(deps.Prompter, resolvers.GetDetectedTargets())
		if err != nil {
			return err
		}
	}

	if len(selectedTargets) > 0 {
		if len(currentIntent.Targets) == 0 || opts.AllSkills {
			action.SetTargets = selectedTargets
		} else {
			reconcileOpts.TargetScope = selectedTargets
		}
	}

	newIntent := core.MergeIntent(currentIntent, action)

	if opts.AllSkills {
		reconcileOpts.UpdateMode = true
	} else if len(effectiveScope) > 0 {
		// Scope to exactly what the user selected
		reconcileOpts.SkillScope = effectiveScope
	}

	// --- Reconcile ---
	result, err := core.Reconcile(ctx, lockfile, namespaceName, sourceURL, fetchResult, manifest, newIntent, reconcileOpts)
	if err != nil {
		return err
	}

	ui.PrintInstallResults(namespaceName, result.Installed, result.Pruned, result.Skipped, result.Conflicts, opts.DryRun)
	return nil
}
