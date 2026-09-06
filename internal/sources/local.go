package sources

import (
	"fmt"
	"path/filepath"

	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
)

func FetchLocal(dirPath string) (string, string, error) {
	expanded := resolvers.ExpandTilde(dirPath)
	resolved, err := filepath.Abs(expanded)
	if err != nil {
		return "", "", utils.NewSourceError("Failed to resolve absolute path", dirPath)
	}

	if !utils.PathExists(resolved) {
		return "", "", utils.NewSourceError("Local directory does not exist", resolved)
	}

	mtime, _ := utils.GetDirectoryMtime(resolved)
	ref := fmt.Sprintf("local-%d", mtime)

	return resolved, ref, nil
}
