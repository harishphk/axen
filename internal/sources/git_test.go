package sources

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func createMockGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Git requires user email and name to commit
	_ = exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	file := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(file, []byte("hello"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}
}

func TestFetchGit(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create mock remote repo
	remoteRepoPath := filepath.Join(tempDir, "remote_repo")
	_ = os.MkdirAll(remoteRepoPath, 0755)
	createMockGitRepo(t, remoteRepoPath)

	// Set AXEN_TEST_HOME to override cache dir
	testHome := filepath.Join(tempDir, "test_home")
	_ = os.MkdirAll(testHome, 0755)
	_ = os.Setenv("AXEN_TEST_HOME", testHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	ctx := context.Background()
	namespace := "my_namespace"

	// Test Clone
	localPath, ref, err := FetchGit(ctx, remoteRepoPath, namespace)
	if err != nil {
		t.Fatalf("FetchGit clone failed: %v", err)
	}

	if localPath == "" {
		t.Fatalf("Expected non-empty local path")
	}

	if ref == "unknown" || ref == "" {
		t.Fatalf("Expected valid git ref, got: %s", ref)
	}

	if _, err := os.Stat(filepath.Join(localPath, ".git")); os.IsNotExist(err) {
		t.Fatalf("Git repository not cloned correctly")
	}

	// Test Pull
	// Make a new commit in the remote repo to pull
	_ = os.WriteFile(filepath.Join(remoteRepoPath, "test2.txt"), []byte("world"), 0644)
	_ = exec.Command("git", "-C", remoteRepoPath, "add", ".").Run()
	_ = exec.Command("git", "-C", remoteRepoPath, "commit", "-m", "second commit").Run()

	localPath2, ref2, err := FetchGit(ctx, remoteRepoPath, namespace)
	if err != nil {
		t.Fatalf("FetchGit pull failed: %v", err)
	}

	if localPath != localPath2 {
		t.Fatalf("Expected local path to be the same, got %s and %s", localPath, localPath2)
	}

	if ref == ref2 {
		t.Fatalf("Expected ref to change after pull, both are %s", ref)
	}

	// Test Error - Clone failure
	badRemote := filepath.Join(tempDir, "does_not_exist")
	_, _, err = FetchGit(ctx, badRemote, "bad_namespace")
	if err == nil {
		t.Fatalf("Expected error when cloning non-existent repo")
	}
	if !strings.Contains(err.Error(), "Git clone failed") {
		t.Fatalf("Expected clone error, got: %v", err)
	}

	// Test Error - Pull failure (simulate by breaking local git repo config)
	_ = exec.Command("rm", "-rf", filepath.Join(localPath, ".git", "config")).Run()
	_, _, err = FetchGit(ctx, remoteRepoPath, namespace)
	if err == nil {
		t.Fatalf("Expected error when pulling broken repo")
	}
	if !strings.Contains(err.Error(), "Git pull failed") {
		t.Fatalf("Expected pull error, got: %v", err)
	}
	
	// Test Error - Cache Dir creation failure
	// We make AXEN_TEST_HOME point to a file instead of a directory
	badHome := filepath.Join(tempDir, "bad_home_file")
	_ = os.WriteFile(badHome, []byte("test"), 0644)
	_ = os.Setenv("AXEN_TEST_HOME", badHome)
	
	_, _, err = FetchGit(ctx, remoteRepoPath, "another_namespace")
	if err == nil {
		t.Fatalf("Expected error when cache dir creation fails")
	}
}
