package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
)

func ReadLockfile() (*models.Lockfile, error) {
	lockfilePath := resolvers.GetLockfilePath()

	if !utils.PathExists(lockfilePath) {
		return models.NewLockfile(), nil
	}

	lockfile, err := utils.ReadJson[models.Lockfile](lockfilePath)
	if err != nil {
		utils.Warn("Lockfile is corrupted, starting fresh")
		return models.NewLockfile(), nil
	}

	// Initialize Namespaces if it was null in JSON
	if lockfile.Namespaces == nil {
		lockfile.Namespaces = make(map[string]models.NamespaceEntry)
	}

	return &lockfile, nil
}

func WriteLockfile(lockfile *models.Lockfile) error {
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return err
	}
	return utils.WriteJson(resolvers.GetLockfilePath(), lockfile)
}

func UpsertNamespace(lockfile *models.Lockfile, name string, entry models.NamespaceEntry) *models.Lockfile {
	lockfile.Namespaces[name] = entry
	return lockfile
}

func RemoveNamespace(lockfile *models.Lockfile, name string) *models.Lockfile {
	delete(lockfile.Namespaces, name)
	return lockfile
}

func RemoveSkillFromLockfile(lockfile *models.Lockfile, namespaceName string, skillName string) *models.Lockfile {
	ns, ok := lockfile.Namespaces[namespaceName]
	if !ok {
		return lockfile
	}

	delete(ns.Skills, skillName)

	if len(ns.Skills) == 0 {
		return RemoveNamespace(lockfile, namespaceName)
	}

	lockfile.Namespaces[namespaceName] = ns
	return lockfile
}

func FindSkillNamespace(lockfile *models.Lockfile, skillName string) (string, *models.NamespaceEntry) {
	for name, entry := range lockfile.Namespaces {
		if _, ok := entry.Skills[skillName]; ok {
			return name, &entry
		}
	}
	return "", nil
}

func GetAllInstalledSkills(lockfile *models.Lockfile) []string {
	var skills []string
	for _, entry := range lockfile.Namespaces {
		for skillName := range entry.Skills {
			skills = append(skills, skillName)
		}
	}
	return skills
}
