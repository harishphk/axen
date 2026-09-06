package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConfigCmd(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tmpDir)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	t.Run("set check_for_updates false", func(t *testing.T) {
		cmd := NewCmdConfig(NewDependencies())
		cmd.SetArgs([]string{"set", "check_for_updates", "false"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("get check_for_updates", func(t *testing.T) {
		cmd := NewCmdConfig(NewDependencies())
		cmd.SetArgs([]string{"get", "check_for_updates"})
		var out bytes.Buffer
		cmd.SetOut(&out)
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(out.String(), "false") {
			t.Fatalf("expected output to contain false, got %s", out.String())
		}
	})

	t.Run("set default_update_policy daily", func(t *testing.T) {
		cmd := NewCmdConfig(NewDependencies())
		cmd.SetArgs([]string{"set", "default_update_policy", "daily"})
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("get default_update_policy", func(t *testing.T) {
		cmd := NewCmdConfig(NewDependencies())
		cmd.SetArgs([]string{"get", "default_update_policy"})
		var out bytes.Buffer
		cmd.SetOut(&out)
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(out.String(), "daily") {
			t.Fatalf("expected output to contain daily, got %s", out.String())
		}
	})
}
