package ui

import "github.com/pterm/pterm"

// Prompter defines the interface for interactive terminal prompts.
// By abstracting prompts, we can inject mock prompters during tests
// so tests don't hang waiting for user input.
type Prompter interface {
	Select(prompt string, options []string) (string, error)
	MultiSelect(prompt string, options []string) ([]string, error)
	InteractiveConfirm(prompt string, printer pterm.InteractiveConfirmPrinter) (bool, error)
}
