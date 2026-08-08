package sources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectSourceType(t *testing.T) {
	cases := []struct {
		source   string
		expected SourceType
	}{
		{"https://github.com/user/repo", SourceTypeGit},
		{"http://github.com/user/repo", SourceTypeGit},
		{"git@github.com:user/repo.git", SourceTypeGit},
		{"/path/to/repo.git", SourceTypeGit},
		{"/local/path/to/source", SourceTypeLocal},
		{"./relative/path", SourceTypeLocal},
		{"C:\\Windows\\Path", SourceTypeLocal}, // fallback to local
	}

	for _, c := range cases {
		result := DetectSourceType(c.source)
		if result != c.expected {
			t.Errorf("DetectSourceType(%q) = %v; expected %v", c.source, result, c.expected)
		}
	}
}

func TestIsGitHubShorthand(t *testing.T) {
	tempDir := t.TempDir()
	localSrc := filepath.Join(tempDir, "local_src")
	_ = os.MkdirAll(localSrc, 0755)

	cases := []struct {
		source   string
		expected bool
	}{
		{"example/agent-skills", true},
		{"owner/repo-name", true},
		{"owner/repo_name", true},
		{"./relative/path", false},
		{"/absolute/path", false},
		{"https://github.com/user/repo", false},
		{"git@github.com:user/repo.git", false},
		{"singleword", false},
		{"owner/repo/extra", false},
		{localSrc, false}, // existing local path
	}

	for _, c := range cases {
		result := IsGitHubShorthand(c.source)
		if result != c.expected {
			t.Errorf("IsGitHubShorthand(%q) = %v; expected %v", c.source, result, c.expected)
		}
	}
}

func TestFetchSource(t *testing.T) {
	tempDir := t.TempDir()

	// Set up AXEN_TEST_HOME for Git fetch caching
	testHome := filepath.Join(tempDir, "test_home")
	_ = os.MkdirAll(testHome, 0755)
	_ = os.Setenv("AXEN_TEST_HOME", testHome)
	_ = os.Setenv("GIT_TERMINAL_PROMPT", "0")
	defer func() {
		_ = os.Unsetenv("AXEN_TEST_HOME")
		_ = os.Unsetenv("GIT_TERMINAL_PROMPT")
	}()

	ctx := context.Background()

	// Test Local
	localSrc := filepath.Join(tempDir, "local_src")
	_ = os.MkdirAll(localSrc, 0755)

	res, err := FetchSource(ctx, localSrc, "ns_local")
	if err != nil {
		t.Fatalf("FetchSource(Local) failed: %v", err)
	}
	if res.Type != SourceTypeLocal {
		t.Fatalf("Expected SourceTypeLocal, got %v", res.Type)
	}

	// Test Git
	gitSrc := filepath.Join(tempDir, "git_src.git") // end with .git to force git detection
	_ = os.MkdirAll(gitSrc, 0755)
	createMockGitRepo(t, gitSrc) // reuse helper from git_test.go

	resGit, err := FetchSource(ctx, gitSrc, "ns_git")
	if err != nil {
		t.Fatalf("FetchSource(Git) failed: %v", err)
	}
	if resGit.Type != SourceTypeGit {
		t.Fatalf("Expected SourceTypeGit, got %v", resGit.Type)
	}

	// Test Git error
	_, err = FetchSource(ctx, filepath.Join(tempDir, "non_existent.git"), "ns_git_err")
	if err == nil {
		t.Fatalf("Expected error for FetchSource Git error case")
	}

	// Test Github Shorthand error
	_, err = FetchSource(ctx, "invaliduser/invalidrepo123", "ns_shorthand_err")
	if err == nil {
		t.Fatalf("Expected error for FetchSource Github shorthand error case")
	}
	if err != nil {
		if !strings.Contains(err.Error(), "Failed to fetch shorthand from GitHub") {
			t.Fatalf("Expected shorthand error message to contain 'Failed to fetch shorthand from GitHub', got: %v", err.Error())
		}
	}

	// Test Local error
	_, err = FetchSource(ctx, filepath.Join(tempDir, "non_existent_local"), "ns_local_err")
	if err == nil {
		t.Fatalf("Expected error for FetchSource Local error case")
	}
}
