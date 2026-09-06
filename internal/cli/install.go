package cli

import (
	"context"
	"fmt"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/services"
	"github.com/harishphk/axen/internal/ui"
	"github.com/harishphk/axen/internal/utils"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type RunInstallOptions struct {
	Force            bool
	DryRun           bool
	AllSkills        bool
	SkillFilter      []string
	TargetFilter     []string
	BundleFilter     []string
	ConflictStrategy string
	SkipAutoDetect   bool
	SkipFetchSpinner bool
}

func NewCmdInstall(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [namespace]",
		Short: "Install skills from a local path or git repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			force, _ := cmd.Flags().GetBool("force")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			allSkills, _ := cmd.Flags().GetBool("all")
			conflictStrategy, _ := cmd.Flags().GetString("conflict-strategy")
			if !models.IsValidConflictStrategy(conflictStrategy) {
				return fmt.Errorf("invalid conflict strategy %q (must be prompt, overwrite, or keep)", conflictStrategy)
			}

			skillsStr, _ := cmd.Flags().GetStringSlice("skills")
			targetsStr, _ := cmd.Flags().GetStringSlice("targets")
			bundleStr, _ := cmd.Flags().GetStringSlice("bundle")

			namespaceName := ""
			if len(args) > 0 {
				namespaceName = args[0]
			}

			opts := RunInstallOptions{
				Force:            force,
				DryRun:           dryRun,
				AllSkills:        allSkills,
				SkillFilter:      skillsStr,
				TargetFilter:     targetsStr,
				BundleFilter:     bundleStr,
				ConflictStrategy: conflictStrategy,
			}

			return runInstall(cmd.Context(), deps, namespaceName, opts)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Overwrite existing skills")
	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().BoolP("all", "a", false, "Install/Sync all skills from the manifest")
	cmd.Flags().StringSliceP("skills", "s", nil, "Comma-separated list of specific skills to install")
	cmd.Flags().StringSliceP("targets", "t", nil, "Comma-separated list of target directories to install into")
	cmd.Flags().StringSliceP("bundle", "b", nil, "Comma-separated list of skill bundles to install")
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
			return fmt.Errorf("source not found: %s", namespaceName)
		}
		sourceURL = nsEntry.Source
	}

	svc := &services.InstallService{}

	var spinner *pterm.SpinnerPrinter
	fetchOpts := services.FetchOptions{}
	if !opts.SkipFetchSpinner {
		fetchOpts.OnFetchStart = func(ns string) {
			spinner, _ = utils.StartSpinner("Fetching " + ns + "...")
		}
		fetchOpts.OnFetchDone = func(ns string, err error) {
			if err != nil {
				spinner.Fail("Failed to fetch")
			}
		}
	}

	fetchResult, manifest, err := svc.FetchManifest(ctx, sourceURL, namespaceName, fetchOpts)
	if err != nil {
		return err
	}
	if !opts.SkipFetchSpinner && spinner != nil {
		spinner.Success(fmt.Sprintf("Fetched %s (%s)", namespaceName, fetchResult.ResolvedSource))
	}

	// Build scope
	var effectiveScope []string
	var addBundles []string
	var addSkills []string
	syncAll := false

	for _, bName := range opts.BundleFilter {
		if _, ok := manifest.Bundles[bName]; !ok {
			utils.Warn("Bundle \"%s\" not found in manifest, skipping", bName)
		}
	}

	if opts.AllSkills {
		syncAll = true
	}

	if len(opts.BundleFilter) == 0 && len(opts.SkillFilter) == 0 && !opts.AllSkills {
		defaultBundleFound := false
		if !opts.SkipAutoDetect {
			for name, b := range manifest.Bundles {
				if b.IsDefault {
					utils.Warn("Auto-installing default bundle: " + name)
					addBundles = []string{name}
					effectiveScope = append(effectiveScope, b.Skills...)
					defaultBundleFound = true
					break
				}
			}
		}

		if !defaultBundleFound {
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
				syncAll = true
			} else if len(selectedBundles) > 0 {
				addBundles = selectedBundles
				for _, bName := range selectedBundles {
					if b, ok := manifest.Bundles[bName]; ok {
						effectiveScope = append(effectiveScope, b.Skills...)
					}
				}
			} else {
				addSkills = selectedSkills
				effectiveScope = append(effectiveScope, selectedSkills...)
			}
		}
	} else {
		effectiveScope = append(effectiveScope, opts.SkillFilter...)
		for _, bName := range opts.BundleFilter {
			if b, ok := manifest.Bundles[bName]; ok {
				effectiveScope = append(effectiveScope, b.Skills...)
			}
		}
		addSkills = opts.SkillFilter
		addBundles = opts.BundleFilter
	}

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

	req := services.InstallRequest{
		NamespaceName:    namespaceName,
		SourceURL:        sourceURL,
		FetchResult:      fetchResult,
		Manifest:         manifest,
		AddSkills:        addSkills,
		AddBundles:       addBundles,
		SetSyncAll:       syncAll,
		SetTargets:       selectedTargets,
		Force:            opts.Force,
		DryRun:           opts.DryRun,
		UpdateMode:       syncAll,
		ConflictStrategy: opts.ConflictStrategy,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
		UntrackedResolver: func(conflicts []core.UntrackedConflict) (map[string]bool, error) {
			return ui.PromptUntrackedConflicts(deps.Prompter, conflicts)
		},
	}
	if !syncAll && len(effectiveScope) > 0 {
		req.SkillScope = effectiveScope
	}
	if len(selectedTargets) > 0 {
		req.TargetScope = selectedTargets
	}

	result, err := svc.Install(ctx, req)
	if err != nil {
		return err
	}

	ui.PrintInstallResults(namespaceName, result.Installed, result.Pruned, result.Skipped, result.Conflicts, opts.DryRun)

	return nil
}
