package cli

import (
	"axen/internal/core"
	"axen/internal/ui"
	"axen/internal/utils"
	"context"
	"fmt"

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
	}

	totalUpdated := 0
	totalUnchanged := 0
	failedCount := 0

	for _, nsName := range toUpdate {
		spinner, _ := utils.StartSpinner("Updating " + nsName + "...")
		nsEntry := lockfile.Namespaces[nsName]

		fetchResult, manifest, err := core.FetchAndResolve(ctx, nsEntry.Source, nsName)
		if err != nil {
			spinner.Fail(err.Error())
			failedCount++
			continue
		}

		if fetchResult.Ref == nsEntry.Ref {
			spinner.Info(nsName + ": already up to date")
			totalUnchanged++
			continue
		}

		intent := core.GetIntent(lockfile, nsName)

		result, err := core.Reconcile(ctx, lockfile, nsName, nsEntry.Source, fetchResult, manifest, intent, core.ReconcileOpts{
			Force:            true,
			DryRun:           dryRun,
			UpdateMode:       true,
			ConflictStrategy: conflictStrategy,
			ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
				return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
			},
		})
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
	return nil
}
