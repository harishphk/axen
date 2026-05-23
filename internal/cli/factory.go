package cli

import (
	"axen/internal/ui"
	"os"
)

// Dependencies holds all injectable components for the CLI commands.
// This allows for isolated unit testing by injecting mock implementations.
type Dependencies struct {
	Prompter ui.Prompter
}

// NewDependencies creates a new Dependencies struct with the default real implementations.
func NewDependencies() *Dependencies {
	if os.Getenv("AXEN_TEST_HOME") != "" {
		return &Dependencies{
			Prompter: &ui.MockPrompter{},
		}
	}
	return &Dependencies{
		Prompter: ui.NewTerminalPrompter(),
	}
}
