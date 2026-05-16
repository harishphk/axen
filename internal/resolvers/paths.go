package resolvers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func GetHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func ExpandTilde(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
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

func GetConfigPath() string {
	return filepath.Join(GetAxenDir(), "axen-config.json")
}

func GetSourcesDir() string {
	return filepath.Join(GetAxenDir(), "sources")
}

func GetCacheDir() string {
	return filepath.Join(GetAxenDir(), "cache")
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
		
		re := regexp.MustCompile(`[/\\]`)
		segments := re.Split(cleaned, -1)
		if len(segments) == 0 {
			return "unknown"
		}
		last := segments[len(segments)-1]
		
		if strings.Contains(last, ":") {
			parts := strings.Split(last, ":")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
			return "unknown"
		}
		if last == "" {
			return "unknown"
		}
		return last
	}
	
	absolutePath, err := filepath.Abs(source)
	if err != nil {
		return "local-skill"
	}
	return filepath.Base(absolutePath)
}
