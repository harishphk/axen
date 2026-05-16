package cli

import (
	"os"
	"testing"
)

func TestRemoveCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Remove empty lockfile (no args)", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"remove"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for empty lockfile")
		}
	})

	t.Run("Remove non-existent namespace", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"remove", "missing-ns", "--all"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("Invalid flags", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"remove", "ns", "--all", "--skills", "skill1"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error when using --all and --skills together")
		}
	})
}
