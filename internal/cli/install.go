package cli

import (
	"axen/internal/core"
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/sources"
	"axen/internal/utils"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [source]",
	Short: "Install skills from a git URL or local path",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		targetsStr, _ := cmd.Flags().GetString("targets")
		skillsStr, _ := cmd.Flags().GetString("skills")

		var targetFilter []string
		if targetsStr != "" {
			for _, t := range strings.Split(targetsStr, ",") {
				targetFilter = append(targetFilter, strings.TrimSpace(t))
			}
		}

		var skillFilter []string
		if skillsStr != "" {
			for _, s := range strings.Split(skillsStr, ",") {
				skillFilter = append(skillFilter, strings.TrimSpace(s))
			}
		}

		namespaceName := resolvers.DeriveNamespace(source)

		if dryRun {
			pterm.Printf("\n%s %s\n\n", pterm.BgCyan.Sprint(pterm.Black(" DRY RUN ")), pterm.Gray("— no changes will be made"))
		}

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Fetching %s...", source))
		fetchResult, err := sources.FetchSource(source, namespaceName)
		if err != nil {
			spinner.Fail("Failed to fetch source: " + err.Error())
			return
		}
		
		refStr := fetchResult.Ref
		if len(refStr) > 8 {
			refStr = refStr[:8]
		}
		spinner.Success(fmt.Sprintf("Fetched %s (%s, ref: %s)", namespaceName, fetchResult.Type, refStr))

		var manifest *models.Manifest
		if core.HasManifest(fetchResult.LocalPath) {
			m, err := core.ReadManifest(fetchResult.LocalPath)
			if err != nil {
				utils.Fatal(err)
				return
			}
			manifest = m
			utils.Info("Using repo manifest: %s skill(s) defined", pterm.Bold.Sprintf("%d", len(manifest.Skills)))
		} else {
			utils.Info("No axen.json found, scanning for skills...")
			scanned, err := core.ScanSkills(fetchResult.LocalPath)
			if err != nil {
				utils.Fatal(err)
				return
			}
			if len(scanned) == 0 {
				utils.Warn("No skills found in source. Nothing to install.")
				return
			}
			manifest = core.GenerateManifest(namespaceName, scanned, targetFilter)

			if !dryRun {
				sourcesDir := filepath.Join(resolvers.GetSourcesDir(), namespaceName)
				utils.EnsureDir(sourcesDir)
				core.WriteManifest(sourcesDir, manifest)
				pterm.Println("  " + pterm.Gray("Auto-generated manifest saved to ~/.axen/sources/"+namespaceName+"/"))
			}
		}

		if len(skillFilter) > 0 {
			filteredSkills := make(map[string]models.SkillEntry)
			for _, skillName := range skillFilter {
				if entry, ok := manifest.Skills[skillName]; ok {
					filteredSkills[skillName] = entry
				} else {
					utils.Warn("Skill %q not found in manifest.", skillName)
				}
			}
			manifest.Skills = filteredSkills

			if len(manifest.Skills) == 0 {
				utils.Error("No matching skills found to install.")
				return
			}
		}

		lockfile, err := core.ReadLockfile()
		if err != nil {
			utils.Fatal(err)
			return
		}

		oldNamespaceState := make(map[string]models.LockfileSkill)
		var oldNamespaceTargets []string
		if nsEntry, ok := lockfile.Namespaces[namespaceName]; ok {
			oldNamespaceState = nsEntry.Skills
			oldNamespaceTargets = nsEntry.Targets
		}

		options := core.InstallOptions{
			Force:   force,
			DryRun:  dryRun,
			Targets: targetFilter,
		}

		results, err := core.InstallSkills(fetchResult.LocalPath, manifest, lockfile, namespaceName, options)
		if err != nil {
			utils.Fatal(err)
			return
		}

		prunes, err := core.PruneSkills(oldNamespaceState, results, oldNamespaceTargets, dryRun, skillFilter)
		if err != nil {
			utils.Fatal(err)
			return
		}

		var installed []core.InstallResult
		var skipped []core.InstallResult
		var conflicts []core.InstallResult

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

		if !dryRun && (len(installed) > 0 || len(prunes) > 0) {
			if force {
				for _, r := range installed {
					ownerNs, _ := core.FindSkillNamespace(lockfile, r.SkillName)
					if ownerNs != "" && ownerNs != namespaceName {
						lockfile = core.RemoveSkillFromLockfile(lockfile, ownerNs, r.SkillName)
					}
				}
			}

			allTargets := resolvers.GetKnownTargets()
			defaultTargets := targetFilter
			if len(defaultTargets) == 0 {
				if len(manifest.Targets) > 0 {
					defaultTargets = manifest.Targets
				} else {
					defaultTargets = allTargets
				}
			}

			recOpts := core.ReconcileOptions{
				DefaultTargets: defaultTargets,
				SkillFilter:    skillFilter,
				OldSkillsState: oldNamespaceState,
			}
			updatedLockfile := core.ReconcileLockfile(lockfile, namespaceName, manifest, results, fetchResult, source, recOpts)
			core.WriteLockfile(updatedLockfile)
		}

		pterm.Println()
		if len(installed) > 0 {
			utils.Success("%s skill(s) installed", pterm.Bold.Sprintf("%d", len(installed)))
			for _, r := range installed {
				for _, d := range r.Destinations {
					pterm.Printf("  %s %s → %s %s\n", pterm.Green("✓"), pterm.Bold.Sprint(r.SkillName), pterm.Gray(d.Path), pterm.Cyan("("+d.Target+")"))
				}
			}
		}

		if len(prunes) > 0 {
			totalRemoved := 0
			for _, p := range prunes {
				totalRemoved += len(p.Removed)
			}
			utils.Warn("%s obsolete skill folder(s) pruned", pterm.Bold.Sprintf("%d", totalRemoved))
			for _, p := range prunes {
				for _, r := range p.Removed {
					pterm.Printf("  %s %s ← %s %s\n", pterm.Red("✗"), pterm.Bold.Sprint(p.SkillName), pterm.Gray(r.Path), pterm.Cyan("("+r.Target+")"))
				}
			}
		}

		if len(skipped) > 0 {
			utils.Warn("%s skill(s) skipped", pterm.Bold.Sprintf("%d", len(skipped)))
		}

		if len(conflicts) > 0 {
			utils.Warn("%s skill(s) had conflicts (use --force to overwrite)", pterm.Bold.Sprintf("%d", len(conflicts)))
		}

		if dryRun {
			pterm.Println(pterm.Gray("\nNo changes made (dry run)."))
		}
	},
}

func init() {
	installCmd.Flags().BoolP("force", "f", false, "Overwrite existing skills on conflict")
	installCmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	installCmd.Flags().StringP("targets", "t", "", "Comma-separated list of targets to install into")
	installCmd.Flags().StringP("skills", "s", "", "Comma-separated list of specific skills to install")
	rootCmd.AddCommand(installCmd)
}
