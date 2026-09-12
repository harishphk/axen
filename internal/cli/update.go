package cli

import (
	"context"
	"fmt"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
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

	var spinner *pterm.SpinnerPrinter
	spec := core.UpdateSpec{
		NamespaceName:    targetNs,
		DryRun:           dryRun,
		ConflictStrategy: conflictStrategy,
		ConflictResolver: func(skillName string, candidates []core.ConflictCandidate) (string, error) {
			return ui.PromptConflictResolution(deps.Prompter, skillName, candidates)
		},
		OnUpdateStart: func(ns string) {
			spinner, _ = utils.StartSpinner("Updating " + ns + "...")
		},
		OnUpdateDone: func(ns string, err error) {
			if err != nil && spinner != nil {
				spinner.Warning(err.Error())
			}
		},
		OnAlreadyUpdated: func(ns string) {
			if spinner != nil {
				spinner.Info(ns + ": already up to date")
			}
		},
		OnUpdated: func(ns string, installedCount, prunedCount int) {
			if spinner != nil {
				msg := pterm.Sprintf("%s: updated %d skill(s)", ns, installedCount)
				if prunedCount > 0 {
					msg += pterm.Sprintf(", pruned %d orphaned skill(s)", prunedCount)
				}
				spinner.Success(msg)
			}
		},
	}

	res, err := deps.Engine.Update(ctx, spec)
	if err != nil && res == nil {
		return err
	}

	if res != nil {
		ui.PrintUpdateSummary(res.TotalUpdated, res.TotalUnchanged, dryRun)

		if len(res.Warnings) > 0 {
			fmt.Println()
			for _, w := range res.Warnings {
				utils.Warn(w)
			}
		}
	}

	return err
}
