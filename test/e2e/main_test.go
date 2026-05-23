package e2e_test

import (
	"axen/internal/cli"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/pterm/pterm"
)

func TestMain(m *testing.M) {
	//nolint:staticcheck
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"axen": axenMain,
	}))
}

func axenMain() int {
	// Freeze the pterm spinner animation to prevent data races during parallel e2e tests
	pterm.DefaultSpinner.Delay = time.Hour
	
	deps := cli.NewDependencies()
	rootCmd := cli.NewRootCmd(deps)
	
	if err := rootCmd.Execute(); err != nil {
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
