package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Update without installation", func(t *testing.T) {
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"update"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error (just warns), got %v", err)
		}
	})

	t.Run("Update non-existent namespace", func(t *testing.T) {
		axenDir := filepath.Join(tmpDir, ".axen")
		_ = os.MkdirAll(axenDir, 0755)
		_ = os.WriteFile(filepath.Join(axenDir, "axen-lock.json"), []byte(`{"axen_version": "1", "namespaces": {"dummy": {"source": "https://example.com/repo.git", "type": "git", "ref": "abc123", "updated_at": "2026-01-01T00:00:00Z", "skills": {}}}}`), 0644)

		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"update", "missing-ns"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for non-existent namespace")
		}
	})
}
