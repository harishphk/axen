package sources

import (
	"axen/internal/utils"
	"fmt"
	"path/filepath"
)

func FetchLocal(dirPath string) (string, string, error) {
	resolved, err := filepath.Abs(dirPath)
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
