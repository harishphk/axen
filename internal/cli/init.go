package cli

import (
	"axen/internal/core"
	"axen/internal/ui"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdInit(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scan for skills and generate/update axen.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			dirStr, _ := cmd.Flags().GetString("dir")

			dir, err := filepath.Abs(dirStr)
			if err != nil {
				return err
			}

			spinner, _ := pterm.DefaultSpinner.Start("Scanning for skills in " + dir + "...")
			scanned, err := core.ScanSkills(dir)
			if err != nil {
				spinner.Fail("Failed to scan skills: " + err.Error())
				return err
			}

			if len(scanned) == 0 {
				spinner.Warning("No skill directories found (looking for folders containing SKILL.md)")
				return nil
			}

			ui.PrintSkillScanResults(scanned)

			if core.HasManifest(dir) {
				pterm.Println("\n" + pterm.Blue("ℹ") + " Updating existing " + pterm.Cyan("axen.json") + "...")
				existing, err := core.ReadManifest(dir)
				if err != nil {
					return err
				}
				merged := core.MergeManifest(existing, scanned)
				if err := core.WriteManifest(dir, merged); err != nil {
					return err
				}
				pterm.Success.Printf("Updated %s with %s skill(s)\n", pterm.Cyan("axen.json"), pterm.Bold.Sprintf("%d", len(merged.Skills)))
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
					return err
				}
				pterm.Success.Printf("Created %s with %s skill(s)\n", pterm.Cyan("axen.json"), pterm.Bold.Sprintf("%d", len(manifest.Skills)))
			}

			return nil
		},
	}
	
	cmd.Flags().StringP("name", "n", "", "Namespace name for the manifest")
	cmd.Flags().StringP("dir", "d", ".", "Directory to scan (default: current)")
	
	return cmd
}
