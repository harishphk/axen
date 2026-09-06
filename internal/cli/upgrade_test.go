package cli

import (
	"os"
	"testing"

	"github.com/harishphk/axen/internal/core"
)

func TestUpgradeCmdFlags(t *testing.T) {
	cmd := NewCmdUpgrade()
	flag := cmd.Flags().Lookup("force")
	if flag == nil {
		t.Fatal("expected --force flag to exist on upgrade command")
	}
	if flag.Shorthand != "f" {
		t.Errorf("expected shorthand 'f', got %q", flag.Shorthand)
	}
}

func TestUpgradeAlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	origVersion := core.Version
	// Set to high version so it's guaranteed to be up to date
	core.Version = "v999.0.0"
	defer func() { core.Version = origVersion }()

	cmd := NewCmdUpgrade()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error when already up to date, got: %v", err)
	}
}
