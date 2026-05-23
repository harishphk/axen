package cli

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	t.Run("Execute help without error", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"--help"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("Verbose flag", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"--verbose", "--help"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
