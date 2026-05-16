package models

import "testing"

func TestNewManifest(t *testing.T) {
	m := NewManifest("my-skill")
	if m.AxenVersion != "1" {
		t.Errorf("Expected version 1, got %s", m.AxenVersion)
	}
	if m.Name != "my-skill" {
		t.Errorf("Expected name my-skill, got %s", m.Name)
	}
	if m.Skills == nil {
		t.Errorf("Skills map not initialized")
	}
}

func TestSkillEntryStruct(t *testing.T) {
	se := SkillEntry{Path: "path"}
	if se.Path != "path" {
		t.Errorf("SkillEntry wrong")
	}
}
