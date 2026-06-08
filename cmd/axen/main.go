package main

import (
	"github.com/harishphk/axen/internal/cli"
	"os"

	"github.com/pterm/pterm"
)

func main() {
	pterm.EnableColor()
	deps := cli.NewDependencies()
	rootCmd := cli.NewRootCmd(deps)

	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Println(err.Error())
		os.Exit(1)
	}
}
