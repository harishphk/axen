package models

type AvailableSkill struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type CacheNamespace struct {
	Available map[string]AvailableSkill `json:"available"`
}

type SourcesCache struct {
	Namespaces map[string]CacheNamespace `json:"namespaces"`
}

func NewSourcesCache() *SourcesCache {
	return &SourcesCache{
		Namespaces: make(map[string]CacheNamespace),
	}
}
