package sources

import (
	"axen/internal/resolvers"
	"axen/internal/utils"
	"os/exec"
	"path/filepath"
	"strings"
)

func FetchGit(url string, namespaceName string) (string, string, error) {
	cacheDir := resolvers.GetCacheDir()
	if err := utils.EnsureDir(cacheDir); err != nil {
		return "", "", err
	}

	localPath := filepath.Join(cacheDir, namespaceName)

	if utils.PathExists(filepath.Join(localPath, ".git")) {
		utils.Debug("Pulling latest for %s...", namespaceName)
		cmd := exec.Command("git", "pull")
		cmd.Dir = localPath
		if err := cmd.Run(); err != nil {
			return "", "", utils.NewSourceError("Git pull failed: "+err.Error(), url)
		}
	} else {
		utils.Debug("Cloning %s...", url)
		cmd := exec.Command("git", "clone", url, localPath)
		if err := cmd.Run(); err != nil {
			return "", "", utils.NewSourceError("Git clone failed: "+err.Error(), url)
		}
	}

	cmd := exec.Command("git", "log", "-1", "--format=%H")
	cmd.Dir = localPath
	out, err := cmd.Output()
	ref := "unknown"
	if err == nil {
		ref = strings.TrimSpace(string(out))
	}

	return localPath, ref, nil
}
