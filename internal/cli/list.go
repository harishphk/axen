package cli

import (
	"axen/internal/core"
	"axen/internal/utils"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Display all installed skills",
	Run: func(cmd *cobra.Command, args []string) {
		lockfile, err := core.ReadLockfile()
		if err != nil {
			utils.Fatal(err)
			return
		}

		if len(lockfile.Namespaces) == 0 {
			utils.Warn("No skills installed. Use `axen install <source>` to get started.")
			return
		}

		totalSkills := 0

		for name, entry := range lockfile.Namespaces {
			sourceLabel := "(local)"
			if entry.Type == "git" {
				sourceLabel = "(" + entry.Source + ")"
			}

			pterm.Printf("\n%s %s\n", pterm.LightCyan(name), pterm.Gray(sourceLabel))
			
			ref := entry.Ref
			if len(ref) > 8 {
				ref = ref[:8]
			}
			updated := strings.Split(entry.UpdatedAt, "T")[0]
			pterm.Println(pterm.Gray("  ref: " + ref + " · updated: " + updated))

			if len(entry.Skills) == 0 {
				pterm.Println(pterm.Gray("  (no skills)"))
				continue
			}

			var skillNames []string
			for s := range entry.Skills {
				skillNames = append(skillNames, s)
			}

			for i, skillName := range skillNames {
				isLast := i == len(skillNames)-1
				prefix := "  ├── "
				if isLast {
					prefix = "  └── "
				}
				pterm.Printf("%s%s\n", pterm.Gray(prefix), pterm.White(skillName))
			}

			totalSkills += len(skillNames)
		}

		pterm.Printf("\n%s skill(s) installed across %s source(s)\n\n", pterm.Bold.Sprintf("%d", totalSkills), pterm.Bold.Sprintf("%d", len(lockfile.Namespaces)))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
