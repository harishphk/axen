package cli

import (
	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/utils"
	"github.com/spf13/cobra"
)

func NewCmdUpgrade() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Axen CLI to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			utils.Info("Checking for Axen updates...")

			res, err := core.UpgradeSelf(cmd.Context(), force, func(msg string) {
				utils.Info("%s", msg)
			})
			if err != nil {
				return err
			}

			if res.IsDevBuild {
				utils.Info("You are running a development build of Axen (dev).")
				utils.Info("The latest official release is %s.", res.NewVersion)
				utils.Info("To replace your local development binary with the official release, run:")
				utils.Info("  axen upgrade --force")
				return nil
			}

			if res.AlreadyUpToDate {
				utils.Success("Axen is already up to date (%s).", res.CurrentVersion)
				return nil
			}

			utils.Success("Axen has been successfully upgraded to %s (%s)!", res.NewVersion, res.ExecutablePath)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force reinstall even if already up to date")

	return cmd
}
