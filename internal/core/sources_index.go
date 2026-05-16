package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
	"path/filepath"
)

func GetSourcesIndexPath() string {
	return filepath.Join(resolvers.GetAxenDir(), "sources.json")
}

func ReadSourcesIndex() (*models.SourcesCache, error) {
	cachePath := GetSourcesIndexPath()

	if !utils.PathExists(cachePath) {
		return models.NewSourcesCache(), nil
	}

	cache, err := utils.ReadJson[models.SourcesCache](cachePath)
	if err != nil {
		utils.Warn("Cache file is corrupted, starting fresh")
		return models.NewSourcesCache(), nil
	}

	if cache.Namespaces == nil {
		cache.Namespaces = make(map[string]models.CacheNamespace)
	}

	for name, entry := range cache.Namespaces {
		if entry.Available == nil {
			entry.Available = make(map[string]models.AvailableSkill)
		}
		cache.Namespaces[name] = entry
	}

	return &cache, nil
}

func WriteSourcesIndex(cache *models.SourcesCache) error {
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return err
	}
	return utils.WriteJson(GetSourcesIndexPath(), cache)
}
