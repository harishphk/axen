package cli

import (
	"axen/internal/core"
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/ui"
	"axen/internal/utils"
	"context"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type RunInstallOptions struct {
	Force            bool
	DryRun           bool
	AllSkills        bool
	TargetFilter     []string
	SkillFilter      []string
	ConflictStrategy string
}

func NewCmdInstall(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [namespace]",
		Short: "Install skills from a registered source or local directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			targetsStr, _ := cmd.Flags().GetStringSlice("targets")
			skillsStr, _ := cmd.Flags().GetStringSlice("skills")
			allFlag, _ := cmd.Flags().GetBool("all")
			conflictStrategy, _ := cmd.Flags().GetString("conflict-strategy")

			opts := RunInstallOptions{
				Force:            force,
				DryRun:           dryRun,
				AllSkills:        allFlag,
				TargetFilter:     targetsStr,
				SkillFilter:      skillsStr,
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

	spinner, _ := pterm.DefaultSpinner.Start("Fetching " + namespaceName + "...")
	fetchResult, manifest, err := core.FetchAndResolve(ctx, sourceURL, namespaceName)
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	spinner.Success("Fetched " + namespaceName)

	syncAll := opts.AllSkills
	if len(opts.SkillFilter) == 0 && !opts.AllSkills {
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

		selectedSkills, isAll, err := ui.PromptSkillSelection(deps.Prompter, namespaceName, newSkillNames, installedCount, len(manifest.Skills))
		if err != nil {
			return err
		}
		if len(selectedSkills) == 0 && !isAll {
			utils.Warn("No skills selected. Aborting.")
			return nil
		}
		opts.SkillFilter = selectedSkills
		if isAll {
			syncAll = true
			opts.SkillFilter = newSkillNames
		}
	}

	if len(opts.TargetFilter) == 0 && !opts.AllSkills {
		targets, err := ui.PromptTargetSelection(deps.Prompter, resolvers.GetDetectedTargets())
		if err != nil {
			return err
		}
		opts.TargetFilter = targets
	}

	installOpts := core.InstallOptions{
		Force:            opts.Force,
		DryRun:           opts.DryRun,
		Targets:          opts.TargetFilter,
		SkillFilter:      opts.SkillFilter,
		ConflictStrategy: opts.ConflictStrategy,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
	}

	results, err := core.InstallSkills(fetchResult.LocalPath, manifest, lockfile, namespaceName, installOpts)
	if err != nil {
		return err
	}

	var oldState map[string]models.LockfileSkill
	if ns, ok := lockfile.Namespaces[namespaceName]; ok {
		oldState = ns.Skills.Installed
	}
	prunes, err := core.PruneSkills(oldState, results, nil, opts.DryRun, opts.SkillFilter)
	if err != nil {
		return err
	}

	var installed, skipped, conflicts []core.InstallResult
	for _, r := range results {
		switch r.Status {
		case "installed":
			installed = append(installed, r)
		case "skipped":
			skipped = append(skipped, r)
		case "conflict":
			conflicts = append(conflicts, r)
		}
	}

	if !opts.DryRun {
		recOpts := core.ReconcileOptions{
			DefaultTargets: opts.TargetFilter,
			SkillFilter:    opts.SkillFilter,
			OldSkillsState: oldState,
			SyncAll:        syncAll,
		}
		updatedLockfile := core.ReconcileLockfile(lockfile, namespaceName, manifest, results, fetchResult, sourceURL, recOpts)
		_ = core.WriteLockfile(updatedLockfile)
	}

	ui.PrintInstallResults(installed, prunes, skipped, conflicts, opts.DryRun)
	return nil
}
