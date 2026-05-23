package models

import "testing"

func TestFrontmatterStruct(t *testing.T) {
	fm := Frontmatter{
		Name:          "name",
		Description:   "desc",
		License:       "MIT",
		Compatibility: "axen>=1",
		Metadata:      map[string]string{"k": "v"},
		AllowedTools:  "tool1",
		Targets:       []string{"t1"},
	}
	if fm.Name != "name" {
		t.Errorf("Struct fields not set correctly")
	}
}
