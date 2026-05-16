package cli

import (
	"axen/internal/utils"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "axen",
	Short:         "Axen - A minimal skill manager for AI agents",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.AddTemplateFunc("StyleHeading", pterm.Cyan)
	cobra.AddTemplateFunc("StyleCommand", pterm.Green)

	usageTmpl := `{{StyleHeading "Usage:"}}{{if .Runnable}}
  {{StyleCommand .UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{StyleCommand .CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{StyleHeading "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{StyleHeading "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{StyleHeading "Available Commands:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{StyleCommand (rpad .Name .NamePadding) }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{StyleHeading "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{StyleHeading "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{StyleHeading "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{StyleCommand (rpad .CommandPath .CommandPathPadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{StyleCommand .CommandPath}} [command] --help" for more information about a command.{{end}}
`
	rootCmd.SetUsageTemplate(usageTmpl)
}

func Execute() {
	pterm.EnableColor()
	if err := rootCmd.Execute(); err != nil {
		if axErr, ok := err.(*utils.AxenError); ok {
			utils.Fatal(axErr)
		} else {
			pterm.Error.Println(err.Error())
		}
		os.Exit(1)
	}
}
