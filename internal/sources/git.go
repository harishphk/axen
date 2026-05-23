package sources

import (
	"axen/internal/resolvers"
	"axen/internal/utils"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

func FetchGit(ctx context.Context, url string, namespaceName string) (string, string, error) {
	sourcesDir := resolvers.GetSourcesDir()
	if err := utils.EnsureDir(sourcesDir); err != nil {
		return "", "", err
	}

	if strings.HasPrefix(url, "-") || strings.HasPrefix(url, "git://-") {
		return "", "", utils.NewSourceError("Invalid or unsafe Git URL provided", url)
	}

	localPath := filepath.Join(sourcesDir, namespaceName)

	if utils.PathExists(filepath.Join(localPath, ".git")) {
		utils.Debug("Pulling latest for %s...", namespaceName)
		cmd := exec.CommandContext(ctx, "git", "pull")
		cmd.Dir = localPath
		if err := cmd.Run(); err != nil {
			return "", "", utils.NewSourceError("Git pull failed: "+err.Error(), url)
		}
	} else {
		utils.Debug("Cloning %s...", url)
		cmd := exec.CommandContext(ctx, "git", "clone", "--", url, localPath)
		if err := cmd.Run(); err != nil {
			return "", "", utils.NewSourceError("Git clone failed: "+err.Error(), url)
		}
	}

	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%H")
	cmd.Dir = localPath
	out, err := cmd.Output()
	ref := "unknown"
	if err == nil {
		ref = strings.TrimSpace(string(out))
	}

	return localPath, ref, nil
}
