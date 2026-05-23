package cli

import (
	"axen/internal/core"
	"axen/internal/utils"
	"fmt"
	"sort"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdList(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display all installed skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			lockfile, err := core.ReadLockfile()
			if err != nil {
				return err
			}

			if len(lockfile.Namespaces) == 0 {
				utils.Warn("No skills installed. Use `axen source add <url>` to get started.")
				return nil
			}

			cache, _ := core.ReadSourcesIndex()
			totalSkills := 0

			// Sort namespace names for deterministic output
			var nsNames []string
			for name := range lockfile.Namespaces {
				nsNames = append(nsNames, name)
			}
			sort.Strings(nsNames)

			for _, name := range nsNames {
				entry := lockfile.Namespaces[name]
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

				if len(entry.Skills.Installed) == 0 {
					pterm.Println(pterm.Gray("  (no skills installed)"))
					if cacheNs, ok := cache.Namespaces[name]; ok && len(cacheNs.Available) > 0 {
						pterm.Println(pterm.Gray(fmt.Sprintf("  %d skill(s) available — run `axen install %s`", len(cacheNs.Available), name)))
					}
					continue
				}

				// Sort skill names for deterministic output
				var skillNames []string
				for s := range entry.Skills.Installed {
					skillNames = append(skillNames, s)
				}
				sort.Strings(skillNames)

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
			return nil
		},
	}
	return cmd
}
