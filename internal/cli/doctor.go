package cli

import (
	"axen/internal/core"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate installation integrity",
	Run: func(cmd *cobra.Command, args []string) {
		issues := 0
		okCount := 0

		lockfile, err := core.ReadLockfile()
		if err != nil {
			utils.Fatal(err)
			return
		}

		if len(lockfile.Namespaces) == 0 {
			pterm.Println(pterm.Yellow("○") + " Lockfile is empty (no skills installed)")
		} else {
			pterm.Println(pterm.Green("✓") + " Lockfile is valid")
			okCount++
		}

		allTargets := resolvers.GetKnownTargets()
		type MissingSkill struct {
			Skill  string
			Target string
			Path   string
		}
		var missingSkills []MissingSkill

		for _, nsEntry := range lockfile.Namespaces {
			nsTargets := nsEntry.Targets
			if len(nsTargets) == 0 {
				nsTargets = allTargets
			}

			for skillName, skillInfo := range nsEntry.Skills {
				skillTargets := skillInfo.Targets
				if len(skillTargets) == 0 {
					skillTargets = nsTargets
				}

				for _, target := range skillTargets {
					targetPath := resolvers.ResolveTargetPath(target)
					if targetPath == nil {
						continue
					}

					skillPath := filepath.Join(*targetPath, skillName)
					if utils.PathExists(*targetPath) && !utils.PathExists(skillPath) {
						missingSkills = append(missingSkills, MissingSkill{Skill: skillName, Target: target, Path: skillPath})
					}
				}
			}
		}

		if len(missingSkills) > 0 {
			for _, m := range missingSkills {
				pterm.Printf("%s Missing: %s should be at %s (%s)\n", pterm.Red("✗"), m.Skill, m.Path, m.Target)
				issues++
			}
		} else if len(lockfile.Namespaces) > 0 {
			pterm.Println(pterm.Green("✓") + " All tracked skills exist on disk")
			okCount++
		}

		trackedSkills := make(map[string]bool)
		for _, nsEntry := range lockfile.Namespaces {
			for skill := range nsEntry.Skills {
				trackedSkills[skill] = true
			}
		}

		type Orphan struct {
			Skill  string
			Target string
			Path   string
		}
		var orphans []Orphan

		for _, target := range allTargets {
			targetPath := resolvers.ResolveTargetPath(target)
			if targetPath == nil || !utils.PathExists(*targetPath) {
				continue
			}

			entries, err := os.ReadDir(*targetPath)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && !trackedSkills[entry.Name()] {
					skillMdPath := filepath.Join(*targetPath, entry.Name(), "SKILL.md")
					skillMdPathLower := filepath.Join(*targetPath, entry.Name(), "skill.md")

					if utils.PathExists(skillMdPath) || utils.PathExists(skillMdPathLower) {
						orphans = append(orphans, Orphan{Skill: entry.Name(), Target: target, Path: filepath.Join(*targetPath, entry.Name())})
					}
				}
			}
		}

		if len(orphans) > 0 {
			for _, o := range orphans {
				pterm.Printf("%s Orphan: %s (%s) — not tracked in lockfile\n", pterm.Yellow("!"), o.Path, o.Target)
				issues++
			}
		} else {
			pterm.Println(pterm.Green("✓") + " No orphan skills detected")
			okCount++
		}

		totalSkills := 0
		for _, ns := range lockfile.Namespaces {
			totalSkills += len(ns.Skills)
		}
		totalNamespaces := len(lockfile.Namespaces)

		pterm.Printf("\n%s namespace(s), %s skill(s) tracked\n", pterm.Bold.Sprintf("%d", totalNamespaces), pterm.Bold.Sprintf("%d", totalSkills))

		if issues > 0 {
			pterm.Printf("\n%s\n", pterm.Red(pterm.Sprintf("%d issue(s) found.", issues)))
		} else {
			pterm.Println("\n" + pterm.Green("✓ Everything looks good!"))
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
