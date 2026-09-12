package core

import (
	"fmt"
	"sort"
	"time"

	"github.com/harishphk/axen/internal/models"
)

// SourceInfo contains presentation-ready summary metadata about a registered source.
type SourceInfo struct {
	Namespace            string
	Source               string
	Type                 string
	UpdatePolicy         string
	InstalledSkillsCount int
	UpdatedAt            string
}

// ListSources returns summary information for all registered sources in the lockfile.
func ListSources() ([]SourceInfo, error) {
	lockfile, err := ReadLockfile()
	if err != nil {
		return nil, err
	}

	var names []string
	for name := range lockfile.Namespaces {
		names = append(names, name)
	}
	sort.Strings(names)

	var sources []SourceInfo
	for _, name := range names {
		entry := lockfile.Namespaces[name]
		sources = append(sources, SourceInfo{
			Namespace:            name,
			Source:               entry.Source,
			Type:                 entry.Type,
			UpdatePolicy:         entry.UpdatePolicy,
			InstalledSkillsCount: len(entry.Skills.Installed),
			UpdatedAt:            entry.UpdatedAt,
		})
	}
	return sources, nil
}

// SourceAddRequest captures parameters for registering a source in the lockfile.
type SourceAddRequest struct {
	NamespaceName string
	SourceURL     string
	SourceType    string
	Ref           string
	UpdatePolicy  string
	Targets       []string
	Manifest      *models.Manifest
}

// AddSource registers a new source namespace in the lockfile and updates the sources cache.
func AddSource(req SourceAddRequest) error {
	lockfile, err := ReadLockfile()
	if err != nil {
		return err
	}

	if _, exists := lockfile.Namespaces[req.NamespaceName]; !exists {
		lockfile.Namespaces[req.NamespaceName] = models.NamespaceEntry{
			Type:         req.SourceType,
			Source:       req.SourceURL,
			Ref:          req.Ref,
			UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
			UpdatePolicy: req.UpdatePolicy,
			Targets:      req.Targets,
			Skills: models.NamespaceSkills{
				Installed: make(map[string]models.LockfileSkill),
			},
		}
		if err := WriteLockfile(lockfile); err != nil {
			return err
		}
	}

	if req.Manifest != nil {
		cache, _ := ReadSourcesIndex()
		if cache == nil {
			cache = models.NewSourcesCache()
		}
		cacheNs := models.CacheNamespace{Available: make(map[string]models.AvailableSkill)}
		for skillName, entry := range req.Manifest.Skills {
			cacheNs.Available[skillName] = models.AvailableSkill{Path: entry.Path, Version: entry.Version}
		}
		cache.Namespaces[req.NamespaceName] = cacheNs
		if err := WriteSourcesIndex(cache); err != nil {
			return err
		}
	}

	return nil
}

// SetSourcePolicy updates the auto-update policy for an existing registered source namespace.
func SetSourcePolicy(namespaceName string, policy string) error {
	lockfile, err := ReadLockfile()
	if err != nil {
		return err
	}

	entry, ok := lockfile.Namespaces[namespaceName]
	if !ok {
		return fmt.Errorf("source %q not found", namespaceName)
	}

	entry.UpdatePolicy = policy
	lockfile.Namespaces[namespaceName] = entry

	return WriteLockfile(lockfile)
}
