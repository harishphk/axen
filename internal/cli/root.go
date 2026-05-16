package cli

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewRootCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "axen",
		Short:         "Axen - A minimal skill manager for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				pterm.EnableDebugMessages()
			}
		},
	}

	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose/debug logging")

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
	cmd.SetUsageTemplate(usageTmpl)

	// Add subcommands
	cmd.AddCommand(NewCmdInstall(deps))
	cmd.AddCommand(NewCmdRemove(deps))
	cmd.AddCommand(NewCmdUpdate(deps))
	cmd.AddCommand(NewCmdSource(deps))
	cmd.AddCommand(NewCmdList(deps))
	cmd.AddCommand(NewCmdDoctor(deps))
	cmd.AddCommand(NewCmdCreate(deps))
	cmd.AddCommand(NewCmdInit(deps))

	return cmd
}
