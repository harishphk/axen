package models

import (
	"runtime"
	"testing"
)

func TestGetDefaultTargets(t *testing.T) {
	// Test unix branch
	goos = "linux"
	targets := GetDefaultTargets()
	if len(targets) != len(DefaultTargetsUnix) {
		t.Errorf("Expected unix targets, got %v", targets)
	}

	// Test windows branch
	goos = "windows"
	targets = GetDefaultTargets()
	if len(targets) != len(DefaultTargetsWindows) {
		t.Errorf("Expected windows targets, got %v", targets)
	}

	// Restore original value
	goos = runtime.GOOS
}

func TestConfigStruct(t *testing.T) {
	dt := "template"
	c := Config{
		Targets:         map[string]string{"a": "b"},
		DefaultTemplate: &dt,
	}
	if c.Targets["a"] != "b" || *c.DefaultTemplate != "template" {
		t.Errorf("Config struct fields not set correctly")
	}
}
