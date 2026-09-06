package cli

import (
	"fmt"
	"strconv"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCmdConfig(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global axen configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}

			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			config, err := core.ReadConfig()
			if err != nil {
				return err
			}

			option, _ := pterm.DefaultInteractiveSelect.
				WithOptions([]string{
					fmt.Sprintf("Check for CLI Updates (Current: %t)", config.CheckForUpdates),
					fmt.Sprintf("Default Update Policy (Current: %s)", config.DefaultUpdatePolicy),
				}).
				Show("Select a configuration option")

			if option == fmt.Sprintf("Check for CLI Updates (Current: %t)", config.CheckForUpdates) {
				result, _ := pterm.DefaultInteractiveConfirm.
					WithDefaultValue(config.CheckForUpdates).
					Show("Check for updates automatically?")
				config.CheckForUpdates = result
			} else if option == fmt.Sprintf("Default Update Policy (Current: %s)", config.DefaultUpdatePolicy) {
				result, _ := pterm.DefaultInteractiveSelect.
					WithOptions(models.ValidUpdatePolicies).
					WithDefaultOption(config.DefaultUpdatePolicy).
					Show("Select default update policy")
				config.DefaultUpdatePolicy = result
			} else {
				return nil
			}

			if err := core.WriteConfig(config); err != nil {
				return err
			}

			pterm.Success.Println("Configuration updated successfully")
			return nil
		},
	}

	cmd.AddCommand(NewCmdConfigGet())
	cmd.AddCommand(NewCmdConfigSet())

	return cmd
}

func NewCmdConfigGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := core.ReadConfig()
			if err != nil {
				return err
			}

			key := args[0]
			switch key {
			case "check_for_updates":
				cmd.Println(config.CheckForUpdates)
			case "default_update_policy":
				cmd.Println(config.DefaultUpdatePolicy)
			default:
				return fmt.Errorf("unknown key: %s", key)
			}
			return nil
		},
	}
}

func NewCmdConfigSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			config, err := core.ReadConfig()
			if err != nil {
				return err
			}

			key := args[0]
			value := args[1]

			switch key {
			case "check_for_updates":
				b, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				config.CheckForUpdates = b
			case "default_update_policy":
				if !models.IsValidUpdatePolicy(value) {
					return fmt.Errorf("invalid update policy: %s", value)
				}
				config.DefaultUpdatePolicy = value
			default:
				return fmt.Errorf("unknown key: %s", key)
			}

			return core.WriteConfig(config)
		},
	}
}
