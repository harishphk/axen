package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPathExists(t *testing.T) {
	tempDir := t.TempDir()

	// Existing file
	existingFile := filepath.Join(tempDir, "exists.txt")
	_ = os.WriteFile(existingFile, []byte("data"), 0644)
	if !PathExists(existingFile) {
		t.Fatal("PathExists returned false for existing file")
	}

	// Existing directory
	if !PathExists(tempDir) {
		t.Fatal("PathExists returned false for existing directory")
	}

	// Non-existent path
	if PathExists(filepath.Join(tempDir, "nope")) {
		t.Fatal("PathExists returned true for non-existent path")
	}

	// Permission error should return false, not true
	// (This was the bug: os.IsNotExist returns false for permission errors,
	// so the old code returned true when it should have returned false.)
	if runtime.GOOS != "windows" {
		noAccessDir := filepath.Join(tempDir, "noaccess")
		_ = EnsureDir(noAccessDir)
		innerFile := filepath.Join(noAccessDir, "inner.txt")
		_ = os.WriteFile(innerFile, []byte("data"), 0644)
		_ = os.Chmod(noAccessDir, 0000)
		defer func() { _ = os.Chmod(noAccessDir, 0755) }() // cleanup


		if PathExists(innerFile) {
			t.Fatal("PathExists returned true for file inside permission-denied directory")
		}
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "nested", "dir")

	err := EnsureDir(target)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !PathExists(target) {
		t.Fatalf("Directory not created: %s", target)
	}

	// Test error case (creating dir over a file)
	fileTarget := filepath.Join(tempDir, "file.txt")
	_ = os.WriteFile(fileTarget, []byte("test"), 0644)
	err = EnsureDir(fileTarget)
	if err == nil {
		t.Fatalf("Expected error when EnsureDir conflicts with file")
	}
}

func TestCopyDir(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src")
	dest := filepath.Join(tempDir, "dest")

	_ = EnsureDir(filepath.Join(src, "subdir"))
	_ = os.WriteFile(filepath.Join(src, "file.txt"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(src, "subdir", "subfile.txt"), []byte("subdata"), 0644)

	err := CopyDir(src, dest)
	if err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	if !PathExists(filepath.Join(dest, "file.txt")) {
		t.Fatalf("file.txt not copied")
	}
	if !PathExists(filepath.Join(dest, "subdir", "subfile.txt")) {
		t.Fatalf("subfile.txt not copied")
	}

	// Test error when src doesn't exist
	err = CopyDir(filepath.Join(tempDir, "missing"), filepath.Join(tempDir, "dest2"))
	if err == nil {
		t.Fatalf("Expected error when src missing")
	}

	// Test EnsureDir error at the beginning
	// destination parent is a file
	badDestFile := filepath.Join(tempDir, "bad_dest_parent.txt")
	_ = os.WriteFile(badDestFile, []byte(""), 0644)
	err = CopyDir(src, filepath.Join(badDestFile, "subdest"))
	if err == nil {
		t.Fatalf("Expected error when dest parent is a file")
	}

	// Test Walk errors (e.g. EnsureDir for subdirectories when creating over file)
	conflictDest := filepath.Join(tempDir, "conflict_dest")
	_ = EnsureDir(conflictDest)
	_ = os.WriteFile(filepath.Join(conflictDest, "subdir"), []byte("not a dir"), 0644)
	err = CopyDir(src, conflictDest)
	if err == nil {
		t.Fatalf("Expected error during Walk when dest subdir conflicts with file")
	}
}

func TestRemoveDir(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "remove_me")
	_ = EnsureDir(target)

	err := RemoveDir(target)
	if err != nil {
		t.Fatalf("RemoveDir failed: %v", err)
	}

	if PathExists(target) {
		t.Fatalf("Directory not removed")
	}

	// Removing non-existent dir should not fail
	err = RemoveDir(filepath.Join(tempDir, "missing"))
	if err != nil {
		t.Fatalf("RemoveDir failed on missing dir: %v", err)
	}
}

type dummyData struct {
	Name string `json:"name"`
}

func TestJsonReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	jsonFile := filepath.Join(tempDir, "data.json")

	data := dummyData{Name: "test"}

	err := WriteJson(jsonFile, data)
	if err != nil {
		t.Fatalf("WriteJson failed: %v", err)
	}

	readData, err := ReadJson[dummyData](jsonFile)
	if err != nil {
		t.Fatalf("ReadJson failed: %v", err)
	}

	if readData.Name != data.Name {
		t.Fatalf("Expected %v, got %v", data, readData)
	}

	// Test read missing
	_, err = ReadJson[dummyData](filepath.Join(tempDir, "missing.json"))
	if err == nil {
		t.Fatalf("Expected error reading missing json")
	}

	// Test read invalid
	badJson := filepath.Join(tempDir, "bad.json")
	_ = os.WriteFile(badJson, []byte("{badjson"), 0644)
	_, err = ReadJson[dummyData](badJson)
	if err == nil {
		t.Fatalf("Expected error parsing bad json")
	}

	// Test write error (e.g., to a dir instead of file)
	err = WriteJson(tempDir, data)
	if err == nil {
		t.Fatalf("Expected error writing json to dir path")
	}

	// Test json.Marshal error
	err = WriteJson(filepath.Join(tempDir, "chan.json"), make(chan int))
	if err == nil {
		t.Fatalf("Expected error when marshaling invalid json data")
	}
}

func TestFindSkillDirs(t *testing.T) {
	tempDir := t.TempDir()

	_ = EnsureDir(filepath.Join(tempDir, "skill1"))
	_ = os.WriteFile(filepath.Join(tempDir, "skill1", "skill.md"), []byte(""), 0644) // case-insensitive check

	_ = EnsureDir(filepath.Join(tempDir, "nested", "skill2"))
	_ = os.WriteFile(filepath.Join(tempDir, "nested", "skill2", "SKILL.md"), []byte(""), 0644)

	_ = EnsureDir(filepath.Join(tempDir, "notaskill"))
	_ = EnsureDir(filepath.Join(tempDir, ".hidden", "skill3"))
	_ = os.WriteFile(filepath.Join(tempDir, ".hidden", "skill3", "SKILL.md"), []byte(""), 0644)

	_ = EnsureDir(filepath.Join(tempDir, "node_modules", "skill4"))
	_ = os.WriteFile(filepath.Join(tempDir, "node_modules", "skill4", "SKILL.md"), []byte(""), 0644)

	dirs, err := FindSkillDirs(tempDir)
	if err != nil {
		t.Fatalf("FindSkillDirs failed: %v", err)
	}

	expected := []string{
		filepath.Join(tempDir, "nested", "skill2"),
		filepath.Join(tempDir, "skill1"),
	}

	// order is not guaranteed by walk, so check if all expected exist and lengths match
	if len(dirs) != len(expected) {
		t.Fatalf("Expected %d skill dirs, got %d: %v", len(expected), len(dirs), dirs)
	}

	foundMap := make(map[string]bool)
	for _, d := range dirs {
		foundMap[d] = true
	}

	for _, e := range expected {
		if !foundMap[e] {
			t.Fatalf("Expected dir not found: %s", e)
		}
	}
    
	// Test error case (unreadable dir to trigger read error)
	// We'll skip this on some platforms if Chmod doesn't work, but it usually helps coverage
	unreadableDir := filepath.Join(tempDir, "unreadable_skills")
	_ = EnsureDir(unreadableDir)
	_ = os.Chmod(unreadableDir, 0000)
	_, _ = FindSkillDirs(unreadableDir) // might err, just covering the branch
	_ = os.Chmod(unreadableDir, 0755)

    // Test error case
    _, err = FindSkillDirs(filepath.Join(tempDir, "does-not-exist"))
    if err == nil {
        t.Fatalf("Expected error for non-existent root dir")
    }
}

func TestGetDirectoryMtime(t *testing.T) {
	tempDir := t.TempDir()

	_ = EnsureDir(filepath.Join(tempDir, "dir1"))
	f1 := filepath.Join(tempDir, "dir1", "file1.txt")
	_ = os.WriteFile(f1, []byte(""), 0644)

	_ = EnsureDir(filepath.Join(tempDir, ".hidden"))
	_ = os.WriteFile(filepath.Join(tempDir, ".hidden", "file2.txt"), []byte(""), 0644)

	mtime, err := GetDirectoryMtime(tempDir)
	if err != nil {
		t.Fatalf("GetDirectoryMtime failed: %v", err)
	}

	if mtime == 0 {
		t.Fatalf("Expected mtime > 0")
	}

	stat, _ := os.Stat(f1)
	expectedMtime := stat.ModTime().UnixMilli()

	if mtime != expectedMtime {
		t.Fatalf("Expected mtime %d, got %d", expectedMtime, mtime)
	}
}
