package utils

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)


func EnsureDir(dirPath string) error {
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return NewFileSystemError("Failed to create directory", dirPath)
	}
	return nil
}

func CopyDir(src, dest string) error {
	if err := EnsureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return EnsureDir(destPath)
		}

		if info.Mode()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// Security: resolve the symlink to an absolute path and ensure it
			// stays within the source directory. This prevents a malicious skill
			// repo from escaping its sandbox via a symlink (e.g. "evil" -> /etc/passwd).
			resolvedTarget := target
			if !filepath.IsAbs(target) {
				resolvedTarget = filepath.Join(filepath.Dir(path), target)
			}
			resolvedTarget = filepath.Clean(resolvedTarget)
			cleanSrc := filepath.Clean(src)
			if !strings.HasPrefix(resolvedTarget, cleanSrc+string(filepath.Separator)) && resolvedTarget != cleanSrc {
				return fmt.Errorf("symlink %q escapes skill directory (target: %q): skipping for safety", path, target)
			}
			return os.Symlink(target, destPath)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}

func RemoveDir(dirPath string) error {
	if PathExists(dirPath) {
		if err := os.RemoveAll(dirPath); err != nil {
			return NewFileSystemError("Failed to remove", dirPath)
		}
	}
	return nil
}

func PathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func ReadJson[T any](filePath string) (T, error) {
	var result T
	data, err := os.ReadFile(filePath)
	if err != nil {
		return result, NewFileSystemError("Failed to read JSON", filePath)
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, NewFileSystemError("Failed to parse JSON", filePath)
	}
	return result, nil
}

func WriteJson(filePath string, data any) error {
	if err := EnsureDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return NewFileSystemError("Failed to write JSON temp file", tmpPath)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return NewFileSystemError("Failed to atomically rename JSON file", filePath)
	}
	return nil
}

func FindSkillDirs(rootDir string) ([]string, error) {
	var skillDirs []string
	
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		hasSkillMd := false
		for _, e := range entries {
			if !e.IsDir() && strings.ToUpper(e.Name()) == "SKILL.MD" {
				hasSkillMd = true
				break
			}
		}

		if hasSkillMd {
			skillDirs = append(skillDirs, dir)
			return nil
		}

		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules" {
				if err := walk(filepath.Join(dir, e.Name())); err != nil {
					return err
				}
			}
		}
		return nil
	}

	err := walk(rootDir)
	return skillDirs, err
}

func GetDirectoryMtime(rootDir string) (int64, error) {
	var maxMtime int64 = 0

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors
		}
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules") && path != rootDir {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			if m := info.ModTime().UnixMilli(); m > maxMtime {
				maxMtime = m
			}
		}
		return nil
	})

	return maxMtime, err
}
