package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"os"
	"testing"
)

func TestLockfileManager(t *testing.T) {
	tempHome := t.TempDir()
	_ = os.Setenv("AXEN_TEST_HOME", tempHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	// 1. ReadLockfile when file doesn't exist
	lockfile, err := ReadLockfile()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if lockfile.Namespaces == nil {
		t.Fatalf("Expected Namespaces to be initialized")
	}

	// 2. WriteLockfile
	lockfile.Namespaces["test-ns"] = models.NamespaceEntry{
		Source: "test-source",
		Type:   "git",
		Skills: models.NamespaceSkills{
			Installed: map[string]models.LockfileSkill{
				"skill1": {Version: "1.0"},
			},
		},
	}
	err = WriteLockfile(lockfile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 3. ReadLockfile when file exists
	lockfile, err = ReadLockfile()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, ok := lockfile.Namespaces["test-ns"]; !ok {
		t.Fatalf("Expected 'test-ns', got %v", lockfile.Namespaces)
	}

	// 4. Corrupt file to test ReadLockfile error path
	err = os.WriteFile(resolvers.GetLockfilePath(), []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	lockfile, err = ReadLockfile()
	if err != nil {
		t.Fatalf("Expected no error (should start fresh on corruption), got %v", err)
	}
	if len(lockfile.Namespaces) != 0 {
		t.Fatalf("Expected empty lockfile, got %v", lockfile.Namespaces)
	}

	// 5. Test UpsertNamespace
	lockfile = UpsertNamespace(lockfile, "ns1", models.NamespaceEntry{
		Source: "src1",
		Type:   "local",
		Skills: models.NamespaceSkills{
			Installed: map[string]models.LockfileSkill{"s1": {Version: "1.1"}},
		},
	})
	if lockfile.Namespaces["ns1"].Source != "src1" {
		t.Fatalf("Expected ns1 source to be src1")
	}

	// 6. Test FindSkillNamespace
	ns, entry := FindSkillNamespace(lockfile, "s1")
	if ns != "ns1" || entry == nil {
		t.Fatalf("Expected to find s1 in ns1, got %v", ns)
	}
	ns, entry = FindSkillNamespace(lockfile, "s2")
	if ns != "" || entry != nil {
		t.Fatalf("Expected not to find s2")
	}

	// 7. Test GetAllInstalledSkills
	skills := GetAllInstalledSkills(lockfile)
	if len(skills) != 1 || skills[0] != "s1" {
		t.Fatalf("Expected [s1], got %v", skills)
	}

	// 8. Test RemoveSkillFromLockfile
	lockfile = RemoveSkillFromLockfile(lockfile, "ns2", "s1") // non-existent ns
	lockfile = RemoveSkillFromLockfile(lockfile, "ns1", "s1") // removes skill from installed
	// ns1 should NOT be deleted because it retains the namespace for auto-syncing
	if _, ok := lockfile.Namespaces["ns1"]; !ok {
		t.Fatalf("Expected ns1 to remain")
	}

	// Setup for RemoveNamespace with multiple installed skills
	lockfile = UpsertNamespace(lockfile, "ns1", models.NamespaceEntry{
		Source: "src1",
		Type:   "local",
		Skills: models.NamespaceSkills{
			Installed: map[string]models.LockfileSkill{"s1": {}, "s2": {}},
		},
	})
	lockfile = RemoveSkillFromLockfile(lockfile, "ns1", "s1") // removes s1, leaves s2
	if _, ok := lockfile.Namespaces["ns1"].Skills.Installed["s2"]; !ok {
		t.Fatalf("Expected s2 to remain in ns1")
	}
	if _, ok := lockfile.Namespaces["ns1"].Skills.Installed["s1"]; ok {
		t.Fatalf("Expected s1 to be removed from ns1")
	}

	// 9. Test RemoveNamespace
	lockfile = RemoveNamespace(lockfile, "ns1")
	if len(lockfile.Namespaces) != 0 {
		t.Fatalf("Expected empty namespaces after RemoveNamespace")
	}

	// 10. WriteLockfile error path
	err = os.RemoveAll(resolvers.GetAxenDir())
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(resolvers.GetAxenDir(), []byte("file"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = WriteLockfile(lockfile)
	if err == nil {
		t.Fatalf("Expected error when axen dir is a file")
	}
}
