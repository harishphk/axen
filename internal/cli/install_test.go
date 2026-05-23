package cli

import (
	"os"
	"testing"
)

func TestInstallCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Install empty lockfile no args", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"install"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for no sources available")
		}
	})

	t.Run("Install non-existent source", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"install", "missing-ns"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for non-existent source")
		}
	})
}
