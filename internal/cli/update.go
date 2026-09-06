package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/services"
	"github.com/harishphk/axen/internal/ui"
	"github.com/harishphk/axen/internal/utils"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdUpdate(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [namespace]",
		Short: "Update installed skills to latest versions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			conflictStrategy, _ := cmd.Flags().GetString("conflict-strategy")
			if !models.IsValidConflictStrategy(conflictStrategy) {
				return fmt.Errorf("invalid conflict strategy %q (must be prompt, overwrite, or keep)", conflictStrategy)
			}

			namespaceName := ""
			if len(args) > 0 {
				namespaceName = args[0]
			}

			return runUpdate(cmd.Context(), deps, namespaceName, dryRun, conflictStrategy)
		},
	}

	cmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	cmd.Flags().StringP("conflict-strategy", "c", "prompt", "Conflict resolution strategy: prompt, overwrite, keep")

	return cmd
}

func runUpdate(ctx context.Context, deps *Dependencies, targetNs string, dryRun bool, conflictStrategy string) error {
	if dryRun {
		ui.PrintDryRunBanner()
	}

	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	var toUpdate []string
	if targetNs != "" {
		if _, ok := lockfile.Namespaces[targetNs]; !ok {
			return fmt.Errorf("namespace %q not found in lockfile", targetNs)
		}
		toUpdate = append(toUpdate, targetNs)
	} else {
		for ns := range lockfile.Namespaces {
			toUpdate = append(toUpdate, ns)
		}
		sort.Strings(toUpdate)
	}

	totalUpdated := 0
	totalUnchanged := 0
	failedCount := 0
	var warnings []string

	svc := &services.InstallService{}

	for _, nsName := range toUpdate {
		nsEntry := lockfile.Namespaces[nsName]

		var spinner *pterm.SpinnerPrinter
		fetchOpts := services.FetchOptions{
			OnFetchStart: func(ns string) {
				spinner, _ = utils.StartSpinner("Updating " + ns + "...")
			},
			OnFetchDone: func(ns string, err error) {
				if err != nil {
					spinner.Warning(err.Error())
					warnings = append(warnings, fmt.Sprintf("%s: %v", nsName, err))
				}
			},
		}

		fetchResult, manifest, err := svc.FetchManifest(ctx, nsEntry.Source, nsName, fetchOpts)
		if err != nil {
			continue
		}

		if fetchResult.Ref == nsEntry.Ref {
			spinner.Info(nsName + ": already up to date")
			totalUnchanged++
			continue
		}

		req := services.InstallRequest{
			NamespaceName:    nsName,
			SourceURL:        nsEntry.Source,
			FetchResult:      fetchResult,
			Manifest:         manifest,
			UpdateMode:       true,
			Force:            true,
			DryRun:           dryRun,
			ConflictStrategy: conflictStrategy,
			ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
				return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
			},
		}

		result, err := svc.Install(ctx, req)
		if err != nil {
			spinner.Fail(err.Error())
			failedCount++
			continue
		}

		msg := pterm.Sprintf("%s: updated %d skill(s)", nsName, len(result.Installed))
		if len(result.Pruned) > 0 {
			msg += pterm.Sprintf(", pruned %d orphaned skill(s)", len(result.Pruned))
		}
		spinner.Success(msg)
		totalUpdated += len(result.Installed)
	}

	ui.PrintUpdateSummary(totalUpdated, totalUnchanged, dryRun)

	if failedCount > 0 {
		return fmt.Errorf("%d namespace(s) failed to update", failedCount)
	}

	if len(warnings) > 0 {
		fmt.Println()
		for _, w := range warnings {
			utils.Warn(w)
		}
	}

	return nil
}
