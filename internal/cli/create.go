package cli

import (
	"axen/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func defaultSkillMd(name string) string {
	var words []string
	for _, w := range strings.Split(name, "-") {
		if len(w) > 0 {
			words = append(words, strings.ToUpper(w[:1])+w[1:])
		}
	}
	title := strings.Join(words, " ")

	return fmt.Sprintf(`---
name: %s
description: "<TODO: Describe what this skill does and when to use it>"
---

# %s

## Instructions

<!-- Add step-by-step instructions for the agent here -->

## Examples

<!-- Add concrete input/output examples to improve accuracy -->
`, name, title)
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Scaffold a new skill from a template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		template, _ := cmd.Flags().GetString("template")
		dirStr, _ := cmd.Flags().GetString("dir")
		if dirStr == "" {
			dirStr = "."
		}

		outputDir, err := filepath.Abs(filepath.Join(dirStr, name))
		if err != nil {
			utils.Fatal(err)
			return
		}

		nameRegex := regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
		if !nameRegex.MatchString(name) || len(name) > 64 {
			utils.Error("Invalid skill name %q. Must be lowercase kebab-case, 1-64 characters.", name)
			return
		}

		if strings.Contains(name, "--") {
			utils.Error("Invalid skill name %q. Must not contain consecutive hyphens.", name)
			return
		}

		if utils.PathExists(outputDir) {
			utils.Error("Directory %q already exists.", outputDir)
			return
		}

		if template != "" {
			templateDir, err := filepath.Abs(template)
			if err != nil || !utils.PathExists(templateDir) {
				utils.Error("Template not found: %s", template)
				return
			}

			utils.Info("Using template: %s", templateDir)
			if err := utils.CopyDir(templateDir, outputDir); err != nil {
				utils.Fatal(err)
				return
			}

			skillMdPath := filepath.Join(outputDir, "SKILL.md")
			if utils.PathExists(skillMdPath) {
				content, err := os.ReadFile(skillMdPath)
				if err == nil {
					re := regexp.MustCompile(`(?m)^name:\s*.+$`)
					updated := re.ReplaceAllString(string(content), "name: "+name)
					os.WriteFile(skillMdPath, []byte(updated), 0644)
				}
			}
		} else {
			utils.EnsureDir(outputDir)
			utils.EnsureDir(filepath.Join(outputDir, "scripts"))
			utils.EnsureDir(filepath.Join(outputDir, "references"))
			utils.EnsureDir(filepath.Join(outputDir, "assets"))

			os.WriteFile(filepath.Join(outputDir, "SKILL.md"), []byte(defaultSkillMd(name)), 0644)
		}

		utils.Success("Created skill: %s/", pterm.Bold.Sprint(name))
		pterm.Println("  " + pterm.Cyan(name+"/"))
		pterm.Println("  " + pterm.Gray("├──") + " SKILL.md")
		pterm.Println("  " + pterm.Gray("├──") + " scripts/")
		pterm.Println("  " + pterm.Gray("├──") + " references/")
		pterm.Println("  " + pterm.Gray("└──") + " assets/")
		pterm.Printf("\nNext: Edit %s to add your instructions.\n", pterm.Cyan(name+"/SKILL.md"))
	},
}

func init() {
	createCmd.Flags().StringP("template", "t", "", "Path to a custom template directory")
	createCmd.Flags().StringP("dir", "d", "", "Parent directory to create in (default: current)")
	rootCmd.AddCommand(createCmd)
}
