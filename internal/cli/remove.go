package cli

import (
	"axen/internal/core"
	"axen/internal/utils"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [namespace]",
	Short: "Remove a repository or specific skills",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceName := args[0]
		all, _ := cmd.Flags().GetBool("all")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		skillsStr, _ := cmd.Flags().GetString("skills")

		if namespaceName == "." || namespaceName == ".." || strings.TrimSpace(namespaceName) == "" {
			utils.Error("Invalid namespace name: %q", namespaceName)
			return
		}

		if !all && skillsStr == "" {
			utils.Error("Safety check: You must specify what to remove.\n  Use --all to uninstall the entire '%s' repository.\n  Use --skills=skill-a,skill-b to uninstall specific skills.", namespaceName)
			return
		}

		if all && skillsStr != "" {
			utils.Error("Cannot use both --all and --skills flags together.")
			return
		}

		if dryRun {
			pterm.Printf("\n%s %s\n\n", pterm.BgCyan.Sprint(pterm.Black(" DRY RUN ")), pterm.Gray("— no changes will be made"))
		}

		lockfile, err := core.ReadLockfile()
		if err != nil {
			utils.Fatal(err)
			return
		}

		nsEntry, exists := lockfile.Namespaces[namespaceName]
		if !exists {
			utils.Error("Namespace %q not found in lockfile.", namespaceName)
			return
		}

		var skillsToRemove []string
		if all {
			for s := range nsEntry.Skills {
				skillsToRemove = append(skillsToRemove, s)
			}
		} else {
			for _, s := range strings.Split(skillsStr, ",") {
				skillsToRemove = append(skillsToRemove, strings.TrimSpace(s))
			}
		}

		nsTargets := nsEntry.Targets
		totalRemoved := 0
		updatedLockfile := lockfile

		for _, skillName := range skillsToRemove {
			skillInfo, skillExists := nsEntry.Skills[skillName]
			if !skillExists && !all {
				utils.Warn("Skill %q is not installed in namespace %q.", skillName, namespaceName)
				continue
			}

			skillTargets := skillInfo.Targets
			if len(skillTargets) == 0 {
				skillTargets = nsTargets
			}

			removedFrom, err := core.UninstallSkillFromTargets(skillName, skillTargets, dryRun)
			if err != nil {
				utils.Fatal(err)
				return
			}

			totalRemoved += len(removedFrom)

			if !dryRun {
				updatedLockfile = core.RemoveSkillFromLockfile(updatedLockfile, namespaceName, skillName)
			}

			if len(removedFrom) > 0 {
				utils.Success("Removed %s from %s location(s):", pterm.Bold.Sprint(skillName), pterm.Bold.Sprintf("%d", len(removedFrom)))
				for _, r := range removedFrom {
					pterm.Printf("  %s %s %s\n", pterm.Green("✓"), pterm.Gray(r.Path), pterm.Cyan("("+r.Target+")"))
				}
			} else {
				utils.Info("Skill %q had no active target folders.", skillName)
			}
		}

		if !dryRun {
			if updatedNs, ok := updatedLockfile.Namespaces[namespaceName]; ok && len(updatedNs.Skills) == 0 {
				updatedLockfile = core.RemoveNamespace(updatedLockfile, namespaceName)
				utils.Info("Namespace %q is now empty and has been removed from the lockfile.", namespaceName)
			} else if !ok {
				utils.Info("Namespace %q is now empty and has been removed from the lockfile.", namespaceName)
			}
			core.WriteLockfile(updatedLockfile)
		}

		if dryRun {
			pterm.Println(pterm.Gray("\nNo changes made (dry run)."))
		}
	},
}

func init() {
	removeCmd.Flags().BoolP("all", "a", false, "Remove the entire repository and all its skills")
	removeCmd.Flags().BoolP("dry-run", "d", false, "Preview changes without executing")
	removeCmd.Flags().StringP("skills", "s", "", "Comma-separated list of specific skills to remove")
	rootCmd.AddCommand(removeCmd)
}
