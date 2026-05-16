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

var (
	createNameRegex   = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
	templateNameRegex = regexp.MustCompile(`(?m)^name:\s*.+$`)
)

func NewCmdCreate(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Scaffold a new skill from a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			template, _ := cmd.Flags().GetString("template")
			dirStr, _ := cmd.Flags().GetString("dir")
			if dirStr == "" {
				dirStr = "."
			}

			outputDir, err := filepath.Abs(filepath.Join(dirStr, name))
			if err != nil {
				return err
			}

			if !createNameRegex.MatchString(name) || len(name) > 64 {
				return fmt.Errorf("invalid skill name %q. Must be lowercase kebab-case, 1-64 characters", name)
			}

			if strings.Contains(name, "--") {
				return fmt.Errorf("invalid skill name %q. Must not contain consecutive hyphens", name)
			}

			if utils.PathExists(outputDir) {
				return fmt.Errorf("directory %q already exists", outputDir)
			}

			if template != "" {
				templateDir, err := filepath.Abs(template)
				if err != nil || !utils.PathExists(templateDir) {
					return fmt.Errorf("template not found: %s", template)
				}

				utils.Info("Using template: %s", templateDir)
				if err := utils.CopyDir(templateDir, outputDir); err != nil {
					return err
				}

				skillMdPath := filepath.Join(outputDir, "SKILL.md")
				if utils.PathExists(skillMdPath) {
					content, err := os.ReadFile(skillMdPath)
					if err == nil {
						updated := templateNameRegex.ReplaceAllString(string(content), "name: "+name)
						_ = os.WriteFile(skillMdPath, []byte(updated), 0644)
					}
				}
			} else {
				_ = utils.EnsureDir(outputDir)
				_ = utils.EnsureDir(filepath.Join(outputDir, "scripts"))
				_ = utils.EnsureDir(filepath.Join(outputDir, "references"))
				_ = utils.EnsureDir(filepath.Join(outputDir, "assets"))

				_ = os.WriteFile(filepath.Join(outputDir, "SKILL.md"), []byte(defaultSkillMd(name)), 0644)
			}

			utils.Success("Created skill: %s/", pterm.Bold.Sprint(name))
			pterm.Println("  " + pterm.Cyan(name+"/"))
			pterm.Println("  " + pterm.Gray("├──") + " SKILL.md")
			pterm.Println("  " + pterm.Gray("├──") + " scripts/")
			pterm.Println("  " + pterm.Gray("├──") + " references/")
			pterm.Println("  " + pterm.Gray("└──") + " assets/")
			pterm.Printf("\nNext: Edit %s to add your instructions.\n", pterm.Cyan(name+"/SKILL.md"))

			return nil
		},
	}

	cmd.Flags().StringP("template", "t", "", "Path to a custom template directory")
	cmd.Flags().StringP("dir", "d", "", "Parent directory to create in (default: current)")

	return cmd
}
