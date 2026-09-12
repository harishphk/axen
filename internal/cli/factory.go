package cli

import (
	"os"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/ui"
)

// Dependencies holds all injectable components for the CLI commands.
// This allows for isolated unit testing by injecting mock implementations.
type Dependencies struct {
	Prompter ui.Prompter
	Engine   core.Engine
}

// NewDependencies creates a new Dependencies struct with the default real implementations.
func NewDependencies() *Dependencies {
	prompter := ui.Prompter(ui.NewTerminalPrompter())
	if os.Getenv("AXEN_TEST_HOME") != "" {
		prompter = &ui.MockPrompter{}
	}
	return &Dependencies{
		Prompter: prompter,
		Engine:   core.NewEngine(),
	}
}
