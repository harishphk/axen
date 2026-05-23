package cli

import (
	"os"
	"testing"
)

func TestDoctorCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Doctor empty lockfile", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"doctor"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
