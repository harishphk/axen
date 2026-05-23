package ui

import "github.com/pterm/pterm"

type MockPrompter struct{}

func (m *MockPrompter) Select(prompt string, options []string) (string, error) {
	if len(options) > 0 {
		return options[0], nil
	}
	return "", nil
}

func (m *MockPrompter) MultiSelect(prompt string, options []string) ([]string, error) {
	return nil, nil
}

func (m *MockPrompter) InteractiveConfirm(prompt string, printer pterm.InteractiveConfirmPrinter) (bool, error) {
	return printer.DefaultValue, nil
}
