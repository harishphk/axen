package resolvers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func GetHomeDir() string {
	if testHome := os.Getenv("AXEN_TEST_HOME"); testHome != "" {
		return testHome
	}
	home, _ := os.UserHomeDir()
	return home
}

var pathSeparatorRegex = regexp.MustCompile(`[/\\]`)

func ExpandTilde(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") || p == "~" {
		return filepath.Join(GetHomeDir(), p[1:])
	}
	return p
}

func GetAxenDir() string {
	return filepath.Join(GetHomeDir(), ".axen")
}

func GetLockfilePath() string {
	return filepath.Join(GetAxenDir(), "axen-lock.json")
}

func GetUpdateCachePath() string {
	return filepath.Join(GetAxenDir(), "update-cache.json")
}

func GetProcessLockPath() string {
	return filepath.Join(GetAxenDir(), "axen.lock")
}

func GetConfigPath() string {
	return filepath.Join(GetAxenDir(), "config.json")
}

func GetSourcesDir() string {
	return filepath.Join(GetAxenDir(), "sources")
}

func GetStagingDir() string {
	return filepath.Join(GetAxenDir(), "staging")
}

func GetTemplatesDir() string {
	return filepath.Join(GetAxenDir(), "templates")
}

func DeriveNamespace(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "git@") {
		cleaned := strings.TrimSuffix(source, ".git")
		cleaned = strings.TrimRight(cleaned, "/\\")

		segments := pathSeparatorRegex.Split(cleaned, -1)
		last := segments[len(segments)-1]

		if strings.Contains(last, ":") {
			parts := strings.Split(last, ":")
			last = parts[len(parts)-1]
		}

		if len(segments) >= 2 {
			secondLast := segments[len(segments)-2]
			if strings.Contains(secondLast, ":") {
				parts := strings.Split(secondLast, ":")
				secondLast = parts[len(parts)-1]
			}
			if !strings.Contains(secondLast, ".") && secondLast != "" && secondLast != "http:" && secondLast != "https:" {
				return secondLast + "__" + last
			}
		}
		return last
	}

	absolutePath, err := filepath.Abs(source)
	if err != nil {
		return "local-skill"
	}
	
	dir := filepath.Dir(absolutePath)
	base := filepath.Base(absolutePath)
	
	parent := filepath.Base(dir)
	if parent != "" && parent != "." && parent != string(filepath.Separator) {
		return parent + "__" + base
	}
	
	return base
}
