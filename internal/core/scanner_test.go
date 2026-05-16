package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create a valid skill
	validDir := filepath.Join(tempDir, "valid")
	_ = os.MkdirAll(validDir, 0755)
	err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(`---
name: valid-skill
description: "A valid test skill"
targets: ["t1"]
---
Body content`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create a skill with invalid frontmatter
	invalidFmDir := filepath.Join(tempDir, "invalid_fm")
	_ = os.MkdirAll(invalidFmDir, 0755)
	err = os.WriteFile(filepath.Join(invalidFmDir, "skill.md"), []byte(`---
name: : invalid : yaml : [
---`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Create a skill missing name
	missingNameDir := filepath.Join(tempDir, "missing_name")
	_ = os.MkdirAll(missingNameDir, 0755)
	err = os.WriteFile(filepath.Join(missingNameDir, "Skill.md"), []byte(`---
targets: ["t1"]
---`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Create a skill with no frontmatter separator
	noFmDir := filepath.Join(tempDir, "no_fm")
	_ = os.MkdirAll(noFmDir, 0755)
	err = os.WriteFile(filepath.Join(noFmDir, "SKILL.md"), []byte(`Just some text`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 5. Test ParseSkillMd on valid
	fm, body, err := ParseSkillMd(validDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fm.Name != "valid-skill" || body != "Body content" {
		t.Fatalf("Parsed incorrectly: %+v %q", fm, body)
	}

	// 6. Test ParseSkillMd on missing file
	_, _, err = ParseSkillMd(filepath.Join(tempDir, "nonexistent"))
	if err == nil {
		t.Fatalf("Expected error for nonexistent dir")
	}

	// 7. Test ParseSkillMd on invalid frontmatter
	_, _, err = ParseSkillMd(invalidFmDir)
	if err == nil {
		t.Fatalf("Expected error for invalid fm")
	}

	// 8. Test ParseSkillMd on missing name
	_, _, err = ParseSkillMd(missingNameDir)
	if err == nil {
		t.Fatalf("Expected error for missing name")
	}

	// 9. Test ParseSkillMd on no frontmatter separator
	_, _, err = ParseSkillMd(noFmDir)
	if err == nil {
		t.Fatalf("Expected error for no fm")
	}

	// 10. Test ScanSkills
	scanned, err := ScanSkills(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Only the valid skill should be returned by ScanSkills, the others skipped
	// Wait, does utils.FindSkillDirs find them? It depends on its implementation.
	// We'll assume FindSkillDirs returns all these directories because they have SKILL.md/skill.md/Skill.md
	if len(scanned) != 1 {
		t.Fatalf("Expected 1 valid scanned skill, got %d", len(scanned))
	}
	if scanned[0].Frontmatter.Name != "valid-skill" {
		t.Fatalf("Expected valid-skill, got %s", scanned[0].Frontmatter.Name)
	}
}
