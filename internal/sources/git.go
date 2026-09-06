package sources

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
)

func FetchGit(ctx context.Context, url string, namespaceName string) (string, string, error) {
	sourcesDir := resolvers.GetSourcesDir()
	if err := utils.EnsureDir(sourcesDir); err != nil {
		return "", "", err
	}

	if strings.HasPrefix(url, "-") || strings.HasPrefix(url, "git://-") || strings.HasPrefix(url, "ext::") {
		return "", "", utils.NewSourceError("Invalid or unsafe Git URL provided", url)
	}

	localPath := filepath.Join(sourcesDir, namespaceName)

	if utils.PathExists(filepath.Join(localPath, ".git")) {
		utils.Debug("Fetching latest for %s...", namespaceName)
		fetchCmd := exec.CommandContext(ctx, "git", "fetch")
		fetchCmd.Dir = localPath
		if out, err := fetchCmd.CombinedOutput(); err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return "", "", utils.NewSourceError("Git fetch failed: "+msg, url)
		}

		utils.Debug("Resetting to remote branch for %s...", namespaceName)
		resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", "@{u}")
		resetCmd.Dir = localPath
		if out, err := resetCmd.CombinedOutput(); err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return "", "", utils.NewSourceError("Git reset failed: "+msg, url)
		}
	} else {
		// If localPath exists but is not a valid git repo, remove it first
		if utils.PathExists(localPath) {
			_ = utils.RemoveDir(localPath)
		}
		utils.Debug("Cloning %s...", url)
		cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--", url, localPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return "", "", utils.NewSourceError("Git clone failed: "+msg, url)
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
