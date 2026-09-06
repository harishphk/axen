package e2e_test

import (
	"github.com/harishphk/axen/internal/cli"
	"os"
	"path/filepath"
	"testing"

	"github.com/harishphk/axen/internal/utils"
	"github.com/pterm/pterm"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	//nolint:staticcheck
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"axen": axenMain,
	}))
}

func axenMain() int {
	// Override spinner to prevent testscript formatting issues and data races
	utils.StartSpinner = func(text string) (*pterm.SpinnerPrinter, error) {
		s := pterm.DefaultSpinner
		return &s, nil
	}

	deps := cli.NewDependencies()
	rootCmd := cli.NewRootCmd(deps)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		utils.Error(err.Error())
		return 1
	}
	return 0
}

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			// Mock AXEN_TEST_HOME inside the testscript's WorkDir
			env.Setenv("AXEN_TEST_HOME", env.WorkDir)

			// Pre-create some mock target directories so axen detects them as valid installation targets
			_ = os.MkdirAll(filepath.Join(env.WorkDir, ".claude", "skills"), 0755)
			_ = os.MkdirAll(filepath.Join(env.WorkDir, ".cursor", "skills"), 0755)
			_ = os.MkdirAll(filepath.Join(env.WorkDir, ".windsurf", "skills"), 0755)

			return nil
		},
	})
}
