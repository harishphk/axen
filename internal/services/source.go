package services

import (
	"fmt"
	"time"

	"github.com/harishphk/axen/internal/core"
	"github.com/harishphk/axen/internal/models"
)

type SourceService struct{}

type SourceAddRequest struct {
	NamespaceName string
	SourceURL     string
	SourceType    string
	Ref           string
	UpdatePolicy  string
	Targets       []string
	Manifest      *models.Manifest
}

func (s *SourceService) Add(req SourceAddRequest) error {
	lockfile, err := core.ReadLockfile()
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
		if err := core.WriteLockfile(lockfile); err != nil {
			return err
		}
	}

	if req.Manifest != nil {
		cache, _ := core.ReadSourcesIndex()
		if cache != nil {
			cacheNs := models.CacheNamespace{Available: make(map[string]models.AvailableSkill)}
			for skillName, entry := range req.Manifest.Skills {
				cacheNs.Available[skillName] = models.AvailableSkill{Path: entry.Path, Version: entry.Version}
			}
			cache.Namespaces[req.NamespaceName] = cacheNs
			_ = core.WriteSourcesIndex(cache)
		}
	}

	return nil
}

func (s *SourceService) SetPolicy(namespaceName string, policy string) error {
	lockfile, err := core.ReadLockfile()
	if err != nil {
		return err
	}

	entry, ok := lockfile.Namespaces[namespaceName]
	if !ok {
		return fmt.Errorf("source %q not found", namespaceName)
	}

	entry.UpdatePolicy = policy
	lockfile.Namespaces[namespaceName] = entry

	return core.WriteLockfile(lockfile)
}
