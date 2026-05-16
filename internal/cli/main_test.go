package cli

import (
	"os"
	"testing"

	"axen/internal/utils"

	"github.com/pterm/pterm"
)

func TestMain(m *testing.M) {
	// Disable pterm output entirely to prevent data races with spinners during parallel cli tests
	pterm.DisableOutput()

	// Override spinner to not start a goroutine
	utils.StartSpinner = func(text string) (*pterm.SpinnerPrinter, error) {
		s := pterm.DefaultSpinner
		return &s, nil
	}
	os.Exit(m.Run())
}
