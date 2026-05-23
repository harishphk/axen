package cli

import (
	"os"
	"testing"
)

func TestListCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("List empty lockfile", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"list"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
