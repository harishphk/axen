package cli

import (
	"axen/internal/core"
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/sources"
	"axen/internal/utils"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [namespace]",
	Short: "Update installed skills to latest versions",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetNs := ""
		if len(args) > 0 {
			targetNs = args[0]
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if dryRun {
			pterm.Printf("\n%s %s\n\n", pterm.BgCyan.Sprint(pterm.Black(" DRY RUN ")), pterm.Gray("— no changes will be made"))
		}

		initialLockfile, err := core.ReadLockfile()
		if err != nil {
			utils.Fatal(err)
			return
		}

		if len(initialLockfile.Namespaces) == 0 {
			utils.Warn("No skills installed. Nothing to update.")
			return
		}

		var toUpdate []string
		if targetNs != "" {
			if _, ok := initialLockfile.Namespaces[targetNs]; !ok {
				utils.Warn("Namespace %q not found in lockfile.", targetNs)
				return
			}
			toUpdate = append(toUpdate, targetNs)
		} else {
			for ns := range initialLockfile.Namespaces {
				toUpdate = append(toUpdate, ns)
			}
		}

		totalUpdated := 0
		totalUnchanged := 0

		for _, nsName := range toUpdate {
			spinner, _ := pterm.DefaultSpinner.Start("Checking " + nsName + "...")

			lockfile, err := core.ReadLockfile()
			if err != nil {
				spinner.Fail(nsName + ": failed — " + err.Error())
				continue
			}

			nsEntry := lockfile.Namespaces[nsName]

			fetchResult, err := sources.FetchSource(nsEntry.Source, nsName)
			if err != nil {
				spinner.Fail(nsName + ": failed — " + err.Error())
				continue
			}

			if fetchResult.Ref == nsEntry.Ref {
				refStr := fetchResult.Ref
				if len(refStr) > 8 {
					refStr = refStr[:8]
				}
				spinner.Info(nsName + ": already up to date (" + refStr + ")")
				totalUnchanged++
				continue
			}

			oldRef := nsEntry.Ref
			if len(oldRef) > 8 {
				oldRef = oldRef[:8]
			}
			newRef := fetchResult.Ref
			if len(newRef) > 8 {
				newRef = newRef[:8]
			}
			spinner.UpdateText("Updating " + nsName + " (" + oldRef + " → " + newRef + ")...")

			var manifest *models.Manifest
			if core.HasManifest(fetchResult.LocalPath) {
				m, err := core.ReadManifest(fetchResult.LocalPath)
				if err != nil {
					spinner.Fail(nsName + ": failed — " + err.Error())
					continue
				}
				manifest = m
			} else {
				scanned, err := core.ScanSkills(fetchResult.LocalPath)
				if err != nil {
					spinner.Fail(nsName + ": failed — " + err.Error())
					continue
				}
				oldNamespaceTargets := nsEntry.Targets
				manifest = core.GenerateManifest(nsName, scanned, oldNamespaceTargets)

				if !dryRun {
					sourcesDir := filepath.Join(resolvers.GetSourcesDir(), nsName)
					utils.EnsureDir(sourcesDir)
					core.WriteManifest(sourcesDir, manifest)
				}
			}

			options := core.InstallOptions{Force: true, DryRun: dryRun}
			results, err := core.InstallSkills(fetchResult.LocalPath, manifest, lockfile, nsName, options)
			if err != nil {
				spinner.Fail(nsName + ": failed — " + err.Error())
				continue
			}

			oldNamespaceTargets := nsEntry.Targets
			prunes, err := core.PruneSkills(nsEntry.Skills, results, oldNamespaceTargets, dryRun, nil)
			if err != nil {
				spinner.Fail(nsName + ": failed — " + err.Error())
				continue
			}

			var installed []core.InstallResult
			for _, r := range results {
				if r.Status == "installed" {
					installed = append(installed, r)
				}
			}

			if !dryRun && (len(installed) > 0 || len(prunes) > 0) {
				allKnownTargets := resolvers.GetKnownTargets()
				var defaultTargets []string
				if len(manifest.Targets) > 0 {
					defaultTargets = manifest.Targets
				} else if len(nsEntry.Targets) > 0 {
					defaultTargets = nsEntry.Targets
				} else {
					defaultTargets = allKnownTargets
				}

				recOpts := core.ReconcileOptions{DefaultTargets: defaultTargets}
				updatedLockfile := core.ReconcileLockfile(lockfile, nsName, manifest, results, fetchResult, nsEntry.Source, recOpts)
				core.WriteLockfile(updatedLockfile)
			}

			prunedCount := 0
			for _, p := range prunes {
				prunedCount += len(p.Removed)
			}

			msg := pterm.Sprintf("%s: updated %d skill(s)", nsName, len(installed))
			if prunedCount > 0 {
				msg += pterm.Sprintf(", pruned %d obsolete folder(s)", prunedCount)
			}

			spinner.Success(msg)
			totalUpdated += len(installed)
		}

		pterm.Println()
		if totalUpdated > 0 {
			utils.Success("%s skill(s) updated", pterm.Bold.Sprintf("%d", totalUpdated))
		}
		if totalUnchanged > 0 {
			utils.Info("%s source(s) already up to date", pterm.Bold.Sprintf("%d", totalUnchanged))
		}

		if dryRun {
			pterm.Println(pterm.Gray("\nNo changes made (dry run)."))
		}
	},
}

func init() {
	updateCmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	rootCmd.AddCommand(updateCmd)
}
