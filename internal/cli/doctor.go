package cli

import (
	"axen/internal/core"
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type missingSkill struct {
	Skill  string
	Target string
	Path   string
}

type orphanSkill struct {
	Skill  string
	Target string
	Path   string
}

func checkMissingSkills(lockfile *models.Lockfile, allTargets []string) []missingSkill {
	var missing []missingSkill

	for _, nsEntry := range lockfile.Namespaces {
		nsTargets := nsEntry.Targets
		if len(nsTargets) == 0 {
			nsTargets = allTargets
		}

		for skillName, skillInfo := range nsEntry.Skills.Installed {
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
					missing = append(missing, missingSkill{Skill: skillName, Target: target, Path: skillPath})
				}
			}
		}
	}
	return missing
}

func checkOrphanSkills(lockfile *models.Lockfile, allTargets []string) []orphanSkill {
	trackedSkills := make(map[string]bool)
	for _, nsEntry := range lockfile.Namespaces {
		for skill := range nsEntry.Skills.Installed {
			trackedSkills[skill] = true
		}
	}

	var orphans []orphanSkill

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
					orphans = append(orphans, orphanSkill{Skill: entry.Name(), Target: target, Path: filepath.Join(*targetPath, entry.Name())})
				}
			}
		}
	}
	return orphans
}

func NewCmdDoctor(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate installation integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := 0

			lockfile, err := core.ReadLockfile()
			if err != nil {
				return err
			}

			if len(lockfile.Namespaces) == 0 {
				pterm.Println(pterm.Yellow("○") + " Lockfile is empty (no skills installed)")
			} else {
				pterm.Println(pterm.Green("✓") + " Lockfile is valid")
			}

			allTargets := resolvers.GetDetectedTargets()
			
			missingSkills := checkMissingSkills(lockfile, allTargets)
			if len(missingSkills) > 0 {
				for _, m := range missingSkills {
					pterm.Printf("%s Missing: %s should be at %s (%s)\n", pterm.Red("✗"), m.Skill, m.Path, m.Target)
					issues++
				}
			} else if len(lockfile.Namespaces) > 0 {
				pterm.Println(pterm.Green("✓") + " All tracked skills exist on disk")
			}

			orphans := checkOrphanSkills(lockfile, allTargets)
			if len(orphans) > 0 {
				for _, o := range orphans {
					pterm.Printf("%s Orphan: %s (%s) — not tracked in lockfile\n", pterm.Yellow("!"), o.Path, o.Target)
					issues++
				}
			} else {
				pterm.Println(pterm.Green("✓") + " No orphan skills detected")
			}

			totalSkills := 0
			for _, ns := range lockfile.Namespaces {
				totalSkills += len(ns.Skills.Installed)
			}
			totalNamespaces := len(lockfile.Namespaces)

			pterm.Printf("\n%s namespace(s), %s skill(s) tracked\n", pterm.Bold.Sprintf("%d", totalNamespaces), pterm.Bold.Sprintf("%d", totalSkills))

			if issues > 0 {
				pterm.Printf("\n%s\n", pterm.Red(pterm.Sprintf("%d issue(s) found.", issues)))
			} else {
				pterm.Println("\n" + pterm.Green("✓ Everything looks good!"))
			}
			return nil
		},
	}
	return cmd
}
