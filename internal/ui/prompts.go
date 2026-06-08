package ui

import (
	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"fmt"
	"sort"
	"strings"

	"github.com/pterm/pterm"
)

func PromptConflictResolution(p Prompter, skillName string, candidates []core.ConflictCandidate) (string, error) {
	pterm.Println()
	pterm.Warning.Printf("⚠ Conflict Detected: Multiple sources provide the skill %q.\n", skillName)

	var options []string
	var nsNames []string

	for _, c := range candidates {
		label := c.Namespace
		if c.IsInstalled {
			label += " (Already installed)"
		}
		options = append(options, label)
		nsNames = append(nsNames, c.Namespace)
	}

	selectedLabel, err := p.Select("Which source should be the active provider?", options)
	if err != nil {
		return "", err
	}

	var selectedIdx int
	for i, opt := range options {
		if opt == selectedLabel {
			selectedIdx = i
			break
		}
	}

	return nsNames[selectedIdx], nil
}

func PromptNamespaceSelection(p Prompter, lockfile *models.Lockfile, hasLocal bool) (string, error) {
	var options []string
	optionMap := make(map[string]string)

	if hasLocal {
		label := "local (Current Directory)"
		options = append(options, label)
		optionMap[label] = "."
	}

	var nsNames []string
	for ns := range lockfile.Namespaces {
		nsNames = append(nsNames, ns)
	}
	sort.Strings(nsNames)

	for _, ns := range nsNames {
		entry := lockfile.Namespaces[ns]
		label := fmt.Sprintf("%s (%s)", ns, entry.Source)
		options = append(options, label)
		optionMap[label] = ns
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no sources available. Run `axen source add <url>` first")
	} else if len(options) == 1 {
		selected := optionMap[options[0]]
		if selected == "." {
			pterm.Println(pterm.Blue("ℹ") + " Auto-selected local directory as source.")
		} else {
			pterm.Printf("%s Auto-selected source: %s\n", pterm.Blue("ℹ"), pterm.Cyan(selected))
		}
		return selected, nil
	}

	selected, err := p.Select("Which source do you want to interact with?", options)
	if err != nil {
		return "", err
	}
	return optionMap[selected], nil
}

func PromptSkillSelection(p Prompter, namespaceName string, newSkillNames []string, installedCount, totalCount int, bundles map[string]models.Bundle) ([]string, []string, bool, error) {
	sort.Strings(newSkillNames)
	newCount := len(newSkillNames)

	if installedCount > 0 {
		pterm.Printf("  %s %s skill(s) already installed, %s new skill(s) available (out of %d total)\n\n",
			pterm.Green("✓"), pterm.Bold.Sprintf("%d", installedCount),
			pterm.Cyan(fmt.Sprintf("%d", newCount)), totalCount)
	}

	if newCount == 0 {
		return nil, nil, false, nil
	}

	installAllLabel := fmt.Sprintf("Install All New (%d skills)", newCount)

	options := []string{installAllLabel}
	if len(bundles) > 0 {
		options = append(options, "Select Bundles")
	}
	options = append(options, "Select Specific Skills")

	mode, err := p.Select("How do you want to install skills from '"+namespaceName+"'?", options)
	if err != nil {
		return nil, nil, false, err
	}

	if mode == installAllLabel {
		return newSkillNames, nil, true, nil
	}

	if mode == "Select Bundles" {
		var bundleNames []string
		for name, b := range bundles {
			if !b.IsDefault {
				bundleNames = append(bundleNames, name)
			}
		}
		sort.Strings(bundleNames)

		if len(bundleNames) == 0 {
			pterm.Warning.Println("No non-default bundles available.")
			return nil, nil, false, nil
		}

		selectedBundles, err := p.MultiSelect("Select bundles to install", bundleNames)
		if err != nil {
			return nil, nil, false, err
		}

		var finalSkills []string
		for _, bName := range selectedBundles {
			finalSkills = append(finalSkills, bundles[bName].Skills...)
		}
		return finalSkills, selectedBundles, false, nil
	}

	selectedSkills, err := p.MultiSelect(fmt.Sprintf("Select skills to install (%d available)", newCount), newSkillNames)
	if err != nil {
		return nil, nil, false, err
	}

	return selectedSkills, nil, false, nil
}

func PromptTargetSelection(p Prompter, detectedTargets []string) ([]string, error) {
	if len(detectedTargets) == 0 {
		return nil, nil
	}
	sort.Strings(detectedTargets)

	installAllTargetsLabel := fmt.Sprintf("Install to All Detected Targets (%d)", len(detectedTargets))
	options := []string{installAllTargetsLabel, "Select Specific Targets"}

	mode, err := p.Select("Which targets do you want to install into?", options)
	if err != nil {
		return nil, err
	}

	if mode == installAllTargetsLabel {
		return detectedTargets, nil
	}

	selectedTargets, err := p.MultiSelect(fmt.Sprintf("Select targets (%d detected)", len(detectedTargets)), detectedTargets)
	if err != nil {
		return nil, err
	}

	return selectedTargets, nil
}

func PromptRemoveMode(p Prompter, namespaceName string, installedSkills []string, installedBundles []string) (bool, []string, []string, error) {
	sort.Strings(installedSkills)
	sort.Strings(installedBundles)
	options := []string{"Remove All (" + fmt.Sprint(len(installedSkills)) + " skills)"}
	if len(installedBundles) > 0 {
		options = append(options, "Select Bundles")
	}
	options = append(options, "Select Specific Skills")

	mode, err := p.Select("What do you want to remove from '"+namespaceName+"'?", options)
	if err != nil {
		return false, nil, nil, err
	}

	if mode == options[0] {
		return true, installedSkills, installedBundles, nil
	}

	if mode == "Select Bundles" {
		selectedBundles, err := p.MultiSelect("Select bundles to remove", installedBundles)
		if err != nil {
			return false, nil, nil, err
		}
		return false, nil, selectedBundles, nil
	}

	if len(installedSkills) == 0 {
		return false, nil, nil, fmt.Errorf("no skills left to remove in this namespace")
	}

	selectedSkills, err := p.MultiSelect("Select the skills to remove", installedSkills)
	if err != nil {
		return false, nil, nil, err
	}

	return false, selectedSkills, nil, nil
}

func PromptSyncAllRemoval(p Prompter, namespaceName string) (bool, error) {
	pterm.Println()
	pterm.Warning.Printfln("This source (%s) is set to auto-sync all new skills.", namespaceName)

	optExclude := "Keep auto-syncing, but EXCLUDE these removed skills"
	optStop := "STOP auto-syncing. Only track my remaining installed skills"

	options := []string{optExclude, optStop}
	mode, err := p.Select("How should future updates be handled?", options)
	if err != nil {
		return false, err
	}

	return mode == optExclude, nil
}

func PromptSourceRemoval(p Prompter, namespaceName string) (bool, error) {
	pterm.Println()
	pterm.Warning.Printfln("This was the last skill installed from '%s'.", namespaceName)

	result, err := p.InteractiveConfirm("Do you want to completely remove this source repository?", *pterm.DefaultInteractiveConfirm.WithDefaultValue(true))

	return result, err
}

func PromptUntrackedConflicts(p Prompter, conflicts []core.UntrackedConflict) (map[string]bool, error) {
	pterm.Println()
	pterm.Warning.Println("⚠ Conflict Detected: Some skills already exist in your target(s):")
	groups := make(map[string][]string)
	for _, c := range conflicts {
		groups[c.Target] = append(groups[c.Target], c.SkillName)
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
		pterm.Printf("  • %s: %s\n", pterm.Cyan(t), strings.Join(skillLabels, ", "))
	}
	pterm.Println()

	options := []string{
		"Skip all (Keep existing manually created skills)",
		"Overwrite all",
		"Decide for each skill individually",
	}

	selected, err := p.Select("How would you like to handle these existing skills?", options)
	if err != nil {
		return nil, err
	}

	decisions := make(map[string]bool)
	switch selected {
	case "Overwrite all":
		for _, c := range conflicts {
			decisions[c.SkillName+":"+c.Target] = true
		}
	case "Decide for each skill individually":
		for _, c := range conflicts {
			pterm.Println()
			optOverwrite := "Overwrite (replace existing skill)"
			optSkip := "Skip (keep existing skill)"
			choice, err := p.Select(fmt.Sprintf("Would you like to overwrite '%s' in %s?", c.SkillName, c.Target), []string{optSkip, optOverwrite})
			if err != nil {
				return nil, err
			}
			decisions[c.SkillName+":"+c.Target] = (choice == optOverwrite)
		}
	default:
		// "Skip all"
		for _, c := range conflicts {
			decisions[c.SkillName+":"+c.Target] = false
		}
	}

	return decisions, nil
}

