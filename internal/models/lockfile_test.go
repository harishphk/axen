package models

import "testing"

func TestNewLockfile(t *testing.T) {
	lf := NewLockfile()
	if lf.AxenVersion != "1" {
		t.Errorf("Expected version 1, got %s", lf.AxenVersion)
	}
	if lf.Namespaces == nil {
		t.Errorf("Namespaces map not initialized")
	}
}

func TestLockfileStructs(t *testing.T) {
	ls := LockfileSkill{Version: "1.0"}
	if ls.Version != "1.0" {
		t.Errorf("LockfileSkill wrong")
	}
	ne := NamespaceEntry{Source: "src"}
	if ne.Source != "src" {
		t.Errorf("NamespaceEntry wrong")
	}
}
