package resolvers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetHomeDir(t *testing.T) {
	_ = os.Setenv("AXEN_TEST_HOME", "/test/home")
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()
	
	if GetHomeDir() != "/test/home" {
		t.Errorf("Expected /test/home, got %s", GetHomeDir())
	}
	
	_ = os.Unsetenv("AXEN_TEST_HOME")
	home, _ := os.UserHomeDir()
	if GetHomeDir() != home {
		t.Errorf("Expected %s, got %s", home, GetHomeDir())
	}
}

func TestExpandTilde(t *testing.T) {
	_ = os.Setenv("AXEN_TEST_HOME", "/test/home")
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	if ExpandTilde("~/foo") != filepath.Join("/test/home", "foo") {
		t.Errorf("Expected %s, got %s", filepath.Join("/test/home", "foo"), ExpandTilde("~/foo"))
	}
	if ExpandTilde("~") != "/test/home" {
		t.Errorf("Expected /test/home, got %s", ExpandTilde("~"))
	}
	if ExpandTilde("/absolute/path") != "/absolute/path" {
		t.Errorf("Expected /absolute/path, got %s", ExpandTilde("/absolute/path"))
	}
}

func TestDirPaths(t *testing.T) {
	_ = os.Setenv("AXEN_TEST_HOME", "/test/home")
	defer func() { _ = os.Unsetenv("AXEN_TEST_HOME") }()

	base := filepath.Join("/test/home", ".axen")
	if GetAxenDir() != base {
		t.Errorf("GetAxenDir fail")
	}
	if GetLockfilePath() != filepath.Join(base, "axen-lock.json") {
		t.Errorf("GetLockfilePath fail")
	}
	if GetConfigPath() != filepath.Join(base, "axen-config.json") {
		t.Errorf("GetConfigPath fail")
	}
	if GetSourcesDir() != filepath.Join(base, "sources") {
		t.Errorf("GetSourcesDir fail")
	}
	if GetStagingDir() != filepath.Join(base, "staging") {
		t.Errorf("GetStagingDir fail")
	}
	if GetTemplatesDir() != filepath.Join(base, "templates") {
		t.Errorf("GetTemplatesDir fail")
	}
}

func TestDeriveNamespace(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"https://github.com/user/repo.git", "user__repo"},
		{"http://github.com/user/repo", "user__repo"},
		{"git@github.com:user/repo.git", "user__repo"},
		{"git@github.com:user/repo", "user__repo"},
		{"https://github.com/user/repo/", "user__repo"},
		{"git@github.com:user/repo/", "user__repo"},
		{"https://github.com", "github.com"},
		{"https://github.com:port", "port"},
		{"https://", ""},
		{"/local/path/to/skill", "skill"},
		{"../relative/skill", "skill"},
		{"http://test:", ""},
	}

	for _, c := range cases {
		if res := DeriveNamespace(c.in); res != c.out {
			t.Errorf("DeriveNamespace(%s) = %s, want %s", c.in, res, c.out)
		}
	}
}
