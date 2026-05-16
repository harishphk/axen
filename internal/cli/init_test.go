package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Init in empty dir", func(t *testing.T) {
		testDir := t.TempDir()
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"init", "--dir", testDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(testDir, "axen.json")); !os.IsNotExist(err) {
			t.Fatalf("expected no axen.json to be created")
		}
	})

	t.Run("Init with skills", func(t *testing.T) {
		testDir := t.TempDir()
		skillDir := filepath.Join(testDir, "my-skill")
		_ = os.Mkdir(skillDir, 0755)
		_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-skill\ndescription: A test skill\n---\n"), 0644)

		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"init", "--dir", testDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, err := os.Stat(filepath.Join(testDir, "axen.json")); os.IsNotExist(err) {
			t.Fatalf("expected axen.json to be created")
		}
	})

	t.Run("Update existing manifest", func(t *testing.T) {
		testDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(testDir, "axen.json"), []byte(`{"axen_version": "1", "name": "test", "skills": {}}`), 0644)
		skillDir := filepath.Join(testDir, "my-skill")
		_ = os.Mkdir(skillDir, 0755)
		_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-skill\ndescription: A test skill\n---\n"), 0644)

		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"init", "--dir", testDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
