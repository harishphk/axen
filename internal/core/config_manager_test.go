package core

import (
	"github.com/harishphk/axen/internal/resolvers"
	"os"
	"testing"
)

func TestConfigManager(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	// 1. ReadConfig when file doesn't exist
	config, err := ReadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(config.Targets) == 0 {
		t.Fatalf("Expected default targets")
	}

	// 2. WriteConfig
	config.Targets = map[string]string{"my-target": "/tmp"}
	err = WriteConfig(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 3. ReadConfig when file exists
	config, err = ReadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(config.Targets) != 1 || config.Targets["my-target"] != "/tmp" {
		t.Fatalf("Expected 'my-target', got %v", config.Targets)
	}

	// 4. InitConfig when file exists
	config, err = InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(config.Targets) != 1 || config.Targets["my-target"] != "/tmp" {
		t.Fatalf("Expected 'my-target', got %v", config.Targets)
	}

	// 5. Corrupt file to test ReadConfig error path
	err = os.WriteFile(resolvers.GetConfigPath(), []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config, err = ReadConfig()
	if err != nil {
		t.Fatalf("Expected no error (should use defaults on parse error), got %v", err)
	}
	if len(config.Targets) == 0 {
		t.Fatalf("Expected default targets on parse error")
	}

	// 6. InitConfig when file doesn't exist
	err = os.Remove(resolvers.GetConfigPath())
	if err != nil {
		t.Fatal(err)
	}

	config, err = InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(config.Targets) == 0 {
		t.Fatalf("Expected default targets")
	}

	// Make config path un-creatable to test WriteConfig error path
	err = os.RemoveAll(resolvers.GetAxenDir())
	if err != nil {
		t.Fatal(err)
	}
	// Note: Creating a file where a dir is expected to fail EnsureDir
	err = os.WriteFile(resolvers.GetAxenDir(), []byte("file"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = WriteConfig(config)
	if err == nil {
		t.Fatalf("Expected error when axen dir is a file")
	}
}
