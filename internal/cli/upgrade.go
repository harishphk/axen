package cli

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/harishphk/axen/internal/utils"
	"github.com/spf13/cobra"
)

func NewCmdUpgrade() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Axen CLI to the latest version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lock, err := AcquireProcessLock()
			if err != nil {
				return err
			}
			defer lock.Unlock()

			utils.Info("Upgrading Axen CLI...")

			var execCmd *exec.Cmd
			if runtime.GOOS != "windows" {
				execCmd = exec.CommandContext(cmd.Context(), "/bin/sh", "-c", "curl -fsSL https://raw.githubusercontent.com/harishphk/axen/main/install.sh | sh")
			} else {
				execCmd = exec.CommandContext(cmd.Context(), "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "irm https://raw.githubusercontent.com/harishphk/axen/main/install.ps1 | iex")
			}
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			return execCmd.Run()
		},
	}
}
