package resolvers

import (
	"axen/internal/models"
	"axen/internal/utils"
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
	return keys
}

func IsKnownTarget(targetName string) bool {
	targets := GetTargetPaths()
	_, ok := targets[targetName]
	return ok
}

func ResetTargetPathCache() {
	cachedTargetPaths = nil
}
