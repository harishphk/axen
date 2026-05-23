package cli

import (
	"os"
	"testing"
)

func TestSourceCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Source add invalid url", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"source", "add", "https://invalid-url.local/nonexistent"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for invalid url")
		}
	})
}
