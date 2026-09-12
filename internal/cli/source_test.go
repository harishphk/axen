package cli

import (
	"os"
	"testing"

	"github.com/harishphk/axen/internal/core"
)

func TestSourceCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Source list empty", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"source", "list"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error for source list empty, got: %v", err)
		}
	})

	t.Run("Source list with registered sources", func(t *testing.T) {
		deps := NewDependencies()
		err := deps.Engine.AddSource(t.Context(), core.SourceAddRequest{
			NamespaceName: "test-src",
			SourceURL:     "https://github.com/test/repo",
			SourceType:    "git",
			UpdatePolicy:  "daily",
		})
		if err != nil {
			t.Fatalf("AddSource failed: %v", err)
		}

		rootCmd := NewRootCmd(deps)
		rootCmd.SetArgs([]string{"source", "list"})
		err = rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error for source list, got: %v", err)
		}
	})

	t.Run("Source policy update", func(t *testing.T) {
		deps := NewDependencies()
		rootCmd := NewRootCmd(deps)
		rootCmd.SetArgs([]string{"source", "policy", "test-src", "weekly"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error for source policy, got: %v", err)
		}
	})
}
