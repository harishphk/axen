package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("Create skill successfully", func(t *testing.T) {
		testDir := t.TempDir()
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "test-skill", "--dir", testDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		skillDir := filepath.Join(testDir, "test-skill")
		if _, err := os.Stat(skillDir); os.IsNotExist(err) {
			t.Fatalf("expected skill dir to be created")
		}
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); os.IsNotExist(err) {
			t.Fatalf("expected SKILL.md to be created")
		}
	})

	t.Run("Invalid name", func(t *testing.T) {
		testDir := t.TempDir()
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "Invalid_Name", "--dir", testDir})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for invalid name")
		}
	})

	t.Run("Consecutive hyphens", func(t *testing.T) {
		testDir := t.TempDir()
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "test--skill", "--dir", testDir})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for consecutive hyphens")
		}
	})

	t.Run("Already exists", func(t *testing.T) {
		testDir := t.TempDir()
		skillDir := filepath.Join(testDir, "exist-skill")
		_ = os.Mkdir(skillDir, 0755)

		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "exist-skill", "--dir", testDir})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for existing directory")
		}
	})

	t.Run("With template", func(t *testing.T) {
		testDir := t.TempDir()
		templateDir := filepath.Join(testDir, "template")
		_ = os.Mkdir(templateDir, 0755)
		_ = os.WriteFile(filepath.Join(templateDir, "SKILL.md"), []byte("---\nname: my-template\n---\n"), 0644)

		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "from-template", "--dir", testDir, "--template", templateDir})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		skillDir := filepath.Join(testDir, "from-template")
		if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); os.IsNotExist(err) {
			t.Fatalf("expected SKILL.md to be created from template")
		}

		content, _ := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
		if string(content) != "---\nname: from-template\n---\n" {
			t.Fatalf("expected template name to be replaced, got %s", string(content))
		}
	})

	t.Run("Invalid template", func(t *testing.T) {
		testDir := t.TempDir()
		rootCmd := NewRootCmd(NewDependencies())
		rootCmd.SetArgs([]string{"create", "bad-template", "--dir", testDir, "--template", "/does/not/exist"})
		err := rootCmd.Execute()
		if err == nil {
			t.Fatalf("expected error for invalid template")
		}
	})
}
