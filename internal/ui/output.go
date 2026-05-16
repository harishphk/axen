package ui

import (
	"axen/internal/core"
	"axen/internal/utils"
	"sort"
	"strings"

	"github.com/pterm/pterm"
)

func PrintDryRunBanner() {
	pterm.Printf("\n%s %s\n\n", pterm.BgCyan.Sprint(pterm.Black(" DRY RUN ")), pterm.Gray("— no changes will be made"))
}

func PrintInstallResults(namespaceName string, installed []core.InstallResult, prunes []core.PrunedSkill, skipped []core.InstallResult, conflicts []core.InstallResult, isDryRun bool) {
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
		var normalConflictsCount int
		for _, r := range conflicts {
			if r.Status == "conflict" {
				normalConflictsCount++
			}
		}
		if normalConflictsCount > 0 {
			utils.Warn("%s skill(s) had conflicts (use --force to overwrite)", pterm.Bold.Sprintf("%d", normalConflictsCount))
		}
	}

	var untrackedSkipped []struct {
		SkillName string
		Target    string
	}
	for _, r := range installed {
		for _, d := range r.SkippedDestinations {
			untrackedSkipped = append(untrackedSkipped, struct{ SkillName, Target string }{r.SkillName, d.Target})
		}
	}
	for _, r := range conflicts {
		for _, d := range r.SkippedDestinations {
			untrackedSkipped = append(untrackedSkipped, struct{ SkillName, Target string }{r.SkillName, d.Target})
		}
	}

	if len(untrackedSkipped) > 0 {
		pterm.Println()
		pterm.Warning.Println("⚠ Warning: The following skill(s) already exist in your target(s) and were skipped to prevent overwriting:")
		
		groups := make(map[string][]string)
		for _, skip := range untrackedSkipped {
			groups[skip.Target] = append(groups[skip.Target], skip.SkillName)
		}

		var targets []string
		for t := range groups {
			targets = append(targets, t)
		}
		sort.Strings(targets)

		for _, t := range targets {
			skills := groups[t]
			sort.Strings(skills)
			var skillLabels []string
			for _, s := range skills {
				skillLabels = append(skillLabels, pterm.Bold.Sprint(s))
			}
			pterm.Printf("  - %s: %s\n", pterm.Cyan(t), strings.Join(skillLabels, ", "))
		}
		pterm.Println()
		pterm.Info.Println("To force overwrite them, run:")
		if namespaceName == "" || namespaceName == "." {
			pterm.Printf("  axen install --force\n")
		} else {
			pterm.Printf("  axen install %s --force\n", namespaceName)
		}
		pterm.Println()
		pterm.Info.Println("Next steps:")
		pterm.Println("  - Run `axen list` to see all installed skills")
	}

	if isDryRun {
		pterm.Println(pterm.Gray("\nNo changes made (dry run)."))
	}
}

func PrintUpdateSummary(totalUpdated int, totalUnchanged int, isDryRun bool) {
	pterm.Println()
	if totalUpdated > 0 {
		utils.Success("%s skill(s) updated", pterm.Bold.Sprintf("%d", totalUpdated))
	}
	if totalUnchanged > 0 {
		utils.Info("%s source(s) already up to date", pterm.Bold.Sprintf("%d", totalUnchanged))
	}

	if isDryRun {
		pterm.Println(pterm.Gray("\nNo changes made (dry run)."))
	}
}

func PrintRemoveResults(skillName string, removedFrom []core.Destination) {
	if len(removedFrom) > 0 {
		utils.Success("Removed %s from %s location(s):", pterm.Bold.Sprint(skillName), pterm.Bold.Sprintf("%d", len(removedFrom)))
		for _, r := range removedFrom {
			pterm.Printf("  %s %s %s\n", pterm.Green("✓"), pterm.Gray(r.Path), pterm.Cyan("("+r.Target+")"))
		}
	} else {
		utils.Info("Skill %q had no active target folders.", skillName)
	}
}

func PrintSkillScanResults(scanned []core.ScannedSkill) {
	pterm.Success.Printf("Found %s skill(s):\n", pterm.Bold.Sprintf("%d", len(scanned)))
	for _, skill := range scanned {
		pterm.Printf("  %s %s %s\n", pterm.Green("•"), pterm.Bold.Sprint(skill.Frontmatter.Name), pterm.Gray("("+skill.RelativePath+")"))
	}
}
