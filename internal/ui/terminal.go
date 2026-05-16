package ui

import (
	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
)

// TerminalPrompter is the real implementation of Prompter using pterm.
type TerminalPrompter struct{}

func NewTerminalPrompter() *TerminalPrompter {
	return &TerminalPrompter{}
}

func (t *TerminalPrompter) Select(prompt string, options []string) (string, error) {
	return pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithMaxHeight(15).
		WithDefaultText(prompt).
		Show()
}

func (t *TerminalPrompter) MultiSelect(prompt string, options []string) ([]string, error) {
	return pterm.DefaultInteractiveMultiselect.
		WithOptions(options).
		WithMaxHeight(15).
		WithCheckmark(&pterm.Checkmark{Checked: pterm.Green("✓"), Unchecked: " "}).
		WithDefaultText(prompt).
		WithFilter(false).
		WithKeySelect(keys.Space).
		WithKeyConfirm(keys.Enter).
		Show()
}

func (t *TerminalPrompter) InteractiveConfirm(prompt string, printer pterm.InteractiveConfirmPrinter) (bool, error) {
	return printer.WithDefaultText(prompt).Show()
}
