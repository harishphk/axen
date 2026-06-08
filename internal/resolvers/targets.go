package resolvers

import (
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/utils"
	"path/filepath"
	"sort"
	"strings"
)

var cachedTargetPaths map[string]string

func GetTargetPaths() map[string]string {
	if cachedTargetPaths != nil {
		return cachedTargetPaths
	}

	targets := make(map[string]string)
	defaultTargets := models.GetDefaultTargets()
	for k, v := range defaultTargets {
		targets[k] = v
	}

	configPath := GetConfigPath()
	if utils.PathExists(configPath) {
		config, err := utils.ReadJson[models.Config](configPath)
		if err != nil {
			utils.Warn("Failed to parse axen-config.json, using defaults")
		} else {
			for k, v := range config.Targets {
				targets[k] = v
			}
		}
	}

	cachedTargetPaths = targets
	return targets
}

func ResolveTargetPath(targetName string) *string {
	targets := GetTargetPaths()
	path, ok := targets[targetName]
	if !ok {
		return nil
	}
	expanded := ExpandTilde(path)
	return &expanded
}

func GetKnownTargets() []string {
	targets := GetTargetPaths()
	keys := make([]string, 0, len(targets))
	for k := range targets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetDetectedTargets returns only targets whose parent agent directory
// exists on disk. For example, "cursor" (~/.cursor/skills/) is only returned
// if ~/.cursor/ exists, meaning the user actually has Cursor installed.
func GetDetectedTargets() []string {
	targets := GetTargetPaths()
	var detected []string
	for name, path := range targets {
		expanded := ExpandTilde(path)
		// Strip the trailing "skills/" (or last segment) to get the parent agent dir
		parentDir := filepath.Dir(strings.TrimSuffix(expanded, "/"))
		if utils.PathExists(parentDir) {
			detected = append(detected, name)
		}
	}
	sort.Strings(detected)
	return detected
}

func IsKnownTarget(targetName string) bool {
	targets := GetTargetPaths()
	_, ok := targets[targetName]
	return ok
}

func ResetTargetPathCache() {
	cachedTargetPaths = nil
}
