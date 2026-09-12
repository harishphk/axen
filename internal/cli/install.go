package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
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
	CachedManifest   *core.SourceManifest
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
	if opts.DryRun {
		ui.PrintDryRunBanner()
	}

	var lockfile *models.Lockfile

	// --- Resolve namespace ---
	if namespaceName == "" {
		var err error
		lockfile, err = core.ReadLockfile()
		if err != nil {
			return err
		}
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
		if opts.CachedManifest != nil && opts.CachedManifest.FetchResult != nil {
			sourceURL = opts.CachedManifest.FetchResult.ResolvedSource
		} else {
			if lockfile == nil {
				var err error
				lockfile, err = core.ReadLockfile()
				if err != nil {
					return err
				}
			}
			nsEntry, exists := lockfile.Namespaces[namespaceName]
			if !exists {
				return fmt.Errorf("source not found: %s", namespaceName)
			}
			sourceURL = nsEntry.Source
		}
	}

	var sourceManifest *core.SourceManifest
	if opts.CachedManifest != nil {
		sourceManifest = opts.CachedManifest
	} else {
		var spinner *pterm.SpinnerPrinter
		inspectOpts := core.InspectOptions{}
		if !opts.SkipFetchSpinner {
			inspectOpts.OnFetchStart = func(ns string) {
				spinner, _ = utils.StartSpinner("Fetching " + ns + "...")
			}
			inspectOpts.OnFetchDone = func(ns string, err error) {
				if err != nil && spinner != nil {
					spinner.Fail("Failed to fetch")
				}
			}
		}

		var err error
		sourceManifest, err = deps.Engine.Inspect(ctx, sourceURL, namespaceName, inspectOpts)
		if err != nil {
			return err
		}
		if !opts.SkipFetchSpinner && spinner != nil {
			spinner.Success(fmt.Sprintf("Fetched %s (%s)", namespaceName, sourceManifest.FetchResult.ResolvedSource))
		}
	}

	manifest := sourceManifest.Manifest

	var addBundles []string
	var addSkills []string
	syncAll := opts.AllSkills

	for _, bName := range opts.BundleFilter {
		if _, ok := manifest.Bundles[bName]; !ok {
			utils.Warn("Bundle \"%s\" not found in manifest, skipping", bName)
		}
	}

	if len(opts.BundleFilter) == 0 && len(opts.SkillFilter) == 0 && !opts.AllSkills {
		// Check if a default bundle exists to skip interactive prompt
		hasDefaultBundle := !opts.SkipAutoDetect && manifest.HasDefaultBundle()

		if !hasDefaultBundle {
			if lockfile == nil {
				var err error
				lockfile, err = core.ReadLockfile()
				if err != nil {
					return err
				}
			}

			var newSkillNames []string
			entry, nsExists := lockfile.Namespaces[namespaceName]
			for s := range manifest.Skills {
				if nsExists {
					if _, installed := entry.Skills.Installed[s]; installed {
						continue
					}
				}
				newSkillNames = append(newSkillNames, s)
			}
			sort.Strings(newSkillNames)

			installedCount := 0
			if nsExists {
				installedCount = len(entry.Skills.Installed)
			}
			selectedSkills, selectedBundles, isAll, err := ui.PromptSkillSelection(deps.Prompter, namespaceName, newSkillNames, installedCount, len(manifest.Skills), manifest.Bundles)
			if err != nil {
				return err
			}
			if len(selectedSkills) == 0 && len(selectedBundles) == 0 && !isAll {
				utils.Warn("No skills selected. Aborting.")
				return nil
			}

			if isAll {
				syncAll = true
			} else {
				addBundles = selectedBundles
				addSkills = selectedSkills
			}
		}
	} else {
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

	spec := core.InstallSpec{
		NamespaceName:  namespaceName,
		SourceURL:      sourceURL,
		SourceManifest: sourceManifest,
		AddSkills:      addSkills,
		AddBundles:     addBundles,
		SyncAll:        syncAll,
		Targets:        selectedTargets,
		TargetScope:    selectedTargets,
		Force:          opts.Force,
		DryRun:         opts.DryRun,
		UpdateMode:     syncAll,
		SkipAutoDetect: opts.SkipAutoDetect,
		OnDefaultBundleDetected: func(bName string) {
			utils.Warn("Auto-installing default bundle: " + bName)
		},
		ConflictStrategy: opts.ConflictStrategy,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
		UntrackedConflictResolver: func(conflicts []core.UntrackedConflict) (map[string]bool, error) {
			return ui.PromptUntrackedConflicts(deps.Prompter, conflicts)
		},
	}

	result, err := deps.Engine.Install(ctx, spec)
	if err != nil {
		return err
	}

	ui.PrintInstallResults(namespaceName, result.Installed, result.Pruned, result.Skipped, result.Conflicts, opts.DryRun)

	return nil
}
