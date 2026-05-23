package ui

import (
	"axen/internal/core"
	"axen/internal/models"
	"fmt"
	"sort"

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

func PromptSkillSelection(p Prompter, namespaceName string, newSkillNames []string, installedCount, totalCount int) ([]string, bool, error) {
	newCount := len(newSkillNames)

	if installedCount > 0 {
		pterm.Printf("  %s %s skill(s) already installed, %s new skill(s) available (out of %d total)\n\n",
			pterm.Green("✓"), pterm.Bold.Sprintf("%d", installedCount),
			pterm.Cyan(fmt.Sprintf("%d", newCount)), totalCount)
	}

	if newCount == 0 {
		return nil, false, nil
	}

	installAllLabel := fmt.Sprintf("Install All New (%d skills)", newCount)
	options := []string{installAllLabel, "Select Specific Skills"}
	
	mode, err := p.Select("How do you want to install skills from '"+namespaceName+"'?", options)
	if err != nil {
		return nil, false, err
	}

	if mode == installAllLabel {
		return newSkillNames, true, nil
	}

	selectedSkills, err := p.MultiSelect(fmt.Sprintf("Select skills to install (%d available)", newCount), newSkillNames)
	if err != nil {
		return nil, false, err
	}

	return selectedSkills, false, nil
}

func PromptTargetSelection(p Prompter, detectedTargets []string) ([]string, error) {
	if len(detectedTargets) == 0 {
		return nil, nil
	}

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

func PromptRemoveMode(p Prompter, namespaceName string, installedSkills []string) (bool, []string, error) {
	options := []string{"Remove All (" + fmt.Sprint(len(installedSkills)) + " skills)", "Select Specific Skills"}
	mode, err := p.Select("How do you want to remove skills from '"+namespaceName+"'?", options)
	if err != nil {
		return false, nil, err
	}

	if mode == options[0] {
		return true, installedSkills, nil
	}

	if len(installedSkills) == 0 {
		return false, nil, fmt.Errorf("no skills left to remove in this namespace")
	}

	selectedSkills, err := p.MultiSelect("Select the skills to remove", installedSkills)
	if err != nil {
		return false, nil, err
	}

	return false, selectedSkills, nil
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
