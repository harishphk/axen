package sources

import (
	"context"
	"os"
	"path/filepath"
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

func TestFetchSource(t *testing.T) {
	tempDir := t.TempDir()

	// Set up AXEN_TEST_HOME for Git fetch caching
	testHome := filepath.Join(tempDir, "test_home")
	_ = os.MkdirAll(testHome, 0755)
	_ = os.Setenv("AXEN_TEST_HOME", testHome)
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

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

	// Test Local error
	_, err = FetchSource(ctx, filepath.Join(tempDir, "non_existent_local"), "ns_local_err")
	if err == nil {
		t.Fatalf("Expected error for FetchSource Local error case")
	}

	// Test HTTP / Unsupported
	// We'll mock DetectSourceType implicitly by checking what happens if we pass "http://" but
	// Wait, the HTTP source type is defined but DetectSourceType only returns Git or Local right now!
	// Let's look at DetectSourceType again:
	// if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") ... returns SourceTypeGit
	// So SourceTypeHttp is actually unreachable through DetectSourceType!
	// Wait, DetectSourceType:
	// func DetectSourceType(source string) SourceType {
	// 	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "git@") || strings.HasSuffix(source, ".git") {
	// 		return SourceTypeGit
	// 	}
	// 	return SourceTypeLocal
	// }
	// So how can SourceTypeHttp be reached in FetchSource? It can't currently be returned by DetectSourceType.
	// We can't reach the HTTP branch or default branch in FetchSource directly through a normal string because DetectSourceType returns only Git or Local.
	// However, if DetectSourceType is changed in the future, it might. But currently it's unreachable for 100% coverage unless we can force it.
}
