package cli

import (
	"strings"

	"github.com/harishphk/axen/internal/core"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewRootCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "axen",
		Short:         "Axen - A minimal skill manager for AI agents",
		Version:       core.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				pterm.EnableDebugMessages()
			}
			
			cmdPath := cmd.CommandPath()
			// Only display update notifications for public-facing commands, not internal or config ones
			if cmd.Name() != "_internal_check_updates" && cmd.Name() != "config" && !strings.HasPrefix(cmdPath, "axen config") {
				core.CheckAndNotifyUpdates()
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			cmdPath := cmd.CommandPath()
			// Only trigger background auto-updates after the foreground command completes and releases locks
			// Don't trigger if the user ran internal, config, update, remove, upgrade, or install commands
			if cmd.Name() != "_internal_check_updates" && cmd.Name() != "config" && !strings.HasPrefix(cmdPath, "axen config") {
				if !strings.HasPrefix(cmdPath, "axen update") && !strings.HasPrefix(cmdPath, "axen remove") && !strings.HasPrefix(cmdPath, "axen upgrade") && !strings.HasPrefix(cmdPath, "axen install") {
					core.TriggerOpportunisticUpdates(cmd.Context())
				}
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
	cmd.AddCommand(NewCmdUpgrade())
	cmd.AddCommand(NewCmdConfig(deps))

	// Hidden internal command for background update checking
	internalCmd := &cobra.Command{
		Use:    "_internal_check_updates",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			lock, err := TryAcquireProcessLock()
			if err != nil {
				return
			}
			defer lock.Unlock()

			core.PerformBackgroundUpdateChecks()
		},
	}
	cmd.AddCommand(internalCmd)

	return cmd
}
