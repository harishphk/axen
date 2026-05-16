package core

import (
	"axen/internal/models"
	"axen/internal/utils"
	"path/filepath"
)

const ManifestFilename = "axen.json"

func ReadManifest(dirPath string) (*models.Manifest, error) {
	manifestPath := filepath.Join(dirPath, ManifestFilename)

	if !utils.PathExists(manifestPath) {
		return nil, utils.NewManifestError("No " + ManifestFilename + " found in " + dirPath)
	}

	manifest, err := utils.ReadJson[models.Manifest](manifestPath)
	if err != nil {
		return nil, utils.NewManifestError("Invalid " + ManifestFilename + ": " + err.Error())
	}

	if err := models.ValidateManifest(&manifest); err != nil {
		return nil, utils.NewManifestError(err.Error())
	}

	return &manifest, nil
}

func HasManifest(dirPath string) bool {
	return utils.PathExists(filepath.Join(dirPath, ManifestFilename))
}

func GenerateManifest(name string, skills []ScannedSkill, defaultTargets []string) *models.Manifest {
	manifest := models.NewManifest(name)
	if len(defaultTargets) > 0 {
		manifest.Targets = defaultTargets
	}

	for _, skill := range skills {
		version := skill.Frontmatter.Metadata["version"]
		manifest.Skills[skill.Frontmatter.Name] = models.SkillEntry{
			Path:    skill.RelativePath,
			Version: version,
			Targets: skill.Frontmatter.Targets,
		}
	}

	return manifest
}

func MergeManifest(existing *models.Manifest, scanned []ScannedSkill) *models.Manifest {
	merged := &models.Manifest{
		AxenVersion: existing.AxenVersion,
		Name:        existing.Name,
		Targets:     existing.Targets,
		Skills:      make(map[string]models.SkillEntry),
	}

	for k, v := range existing.Skills {
		merged.Skills[k] = v
	}

	scannedNames := make(map[string]bool)

	for _, skill := range scanned {
		name := skill.Frontmatter.Name
		scannedNames[name] = true

		version := skill.Frontmatter.Metadata["version"]

		if existingSkill, ok := merged.Skills[name]; ok {
			if version == "" {
				version = existingSkill.Version
			}
			merged.Skills[name] = models.SkillEntry{
				Path:    skill.RelativePath,
				Version: version,
				Targets: existingSkill.Targets,
			}
		} else {
			merged.Skills[name] = models.SkillEntry{
				Path:    skill.RelativePath,
				Version: version,
			}
			utils.Info("  + New skill discovered: %s", name)
		}
	}

	for name := range merged.Skills {
		if !scannedNames[name] {
			utils.Warn("  ⚠ Skill %q is in manifest but not found on disk", name)
		}
	}

	return merged
}

func WriteManifest(dirPath string, manifest *models.Manifest) error {
	manifestPath := filepath.Join(dirPath, ManifestFilename)
	return utils.WriteJson(manifestPath, manifest)
}
