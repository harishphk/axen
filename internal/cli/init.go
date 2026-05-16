package cli

import (
	"axen/internal/core"
	"axen/internal/utils"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scan for skills and generate/update axen.json",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		dirStr, _ := cmd.Flags().GetString("dir")

		dir, err := filepath.Abs(dirStr)
		if err != nil {
			utils.Fatal(err)
			return
		}

		spinner, _ := pterm.DefaultSpinner.Start("Scanning for skills in " + dir + "...")
		scanned, err := core.ScanSkills(dir)
		if err != nil {
			spinner.Fail("Failed to scan skills: " + err.Error())
			return
		}

		if len(scanned) == 0 {
			spinner.Warning("No skill directories found (looking for folders containing SKILL.md)")
			return
		}

		spinner.Success(pterm.Sprintf("Found %s skill(s):", pterm.Bold.Sprintf("%d", len(scanned))))
		for _, skill := range scanned {
			pterm.Printf("  %s %s %s\n", pterm.Green("•"), pterm.Bold.Sprint(skill.Frontmatter.Name), pterm.Gray("("+skill.RelativePath+")"))
		}

		if core.HasManifest(dir) {
			pterm.Println("\n" + pterm.Blue("ℹ") + " Updating existing " + pterm.Cyan("axen.json") + "...")
			existing, err := core.ReadManifest(dir)
			if err != nil {
				utils.Fatal(err)
				return
			}
			merged := core.MergeManifest(existing, scanned)
			if err := core.WriteManifest(dir, merged); err != nil {
				utils.Fatal(err)
				return
			}
			utils.Success("Updated %s with %s skill(s)", pterm.Cyan("axen.json"), pterm.Bold.Sprintf("%d", len(merged.Skills)))
		} else {
			if name == "" {
				name = filepath.Base(dir)
				if name == "." || name == "" {
					name = "my-skills"
				}
			}
			pterm.Println("\n" + pterm.Blue("ℹ") + " Generating " + pterm.Cyan("axen.json") + " (namespace: \"" + pterm.Green(name) + "\")...")
			manifest := core.GenerateManifest(name, scanned, nil)
			if err := core.WriteManifest(dir, manifest); err != nil {
				utils.Fatal(err)
				return
			}
			utils.Success("Created %s with %s skill(s)", pterm.Cyan("axen.json"), pterm.Bold.Sprintf("%d", len(manifest.Skills)))
		}
	},
}

func init() {
	initCmd.Flags().StringP("name", "n", "", "Namespace name for the manifest")
	initCmd.Flags().StringP("dir", "d", ".", "Directory to scan (default: current)")
	rootCmd.AddCommand(initCmd)
}
