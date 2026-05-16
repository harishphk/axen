package core

import (
	"axen/internal/models"
	"axen/internal/utils"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ScannedSkill struct {
	Frontmatter  models.Frontmatter
	DirPath      string
	RelativePath string
	Body         string
}

func ParseSkillMd(skillDir string) (models.Frontmatter, string, error) {
	possibleNames := []string{"SKILL.md", "skill.md", "Skill.md"}
	var filePath string
	var fileContent string

	for _, name := range possibleNames {
		candidate := filepath.Join(skillDir, name)
		data, err := os.ReadFile(candidate)
		if err == nil {
			filePath = candidate
			fileContent = string(data)
			break
		}
	}

	if filePath == "" || fileContent == "" {
		return models.Frontmatter{}, "", utils.NewFrontmatterError("No SKILL.md file found", skillDir)
	}

	// Simple frontmatter parser: look for ---
	parts := strings.SplitN(fileContent, "---", 3)
	if len(parts) < 3 {
		return models.Frontmatter{}, "", utils.NewFrontmatterError("SKILL.md has no YAML frontmatter", filePath)
	}

	frontmatterStr := parts[1]
	body := strings.TrimSpace(parts[2])

	var frontmatter models.Frontmatter
	err := yaml.Unmarshal([]byte(frontmatterStr), &frontmatter)
	if err != nil {
		return models.Frontmatter{}, "", utils.NewFrontmatterError("Invalid frontmatter: "+err.Error(), filePath)
	}

	// Basic validation
	if frontmatter.Name == "" {
		return models.Frontmatter{}, "", utils.NewFrontmatterError("Invalid frontmatter: missing name", filePath)
	}

	if err := models.ValidateFrontmatter(&frontmatter); err != nil {
		return models.Frontmatter{}, "", utils.NewFrontmatterError(err.Error(), filePath)
	}

	return frontmatter, body, nil
}

func ScanSkills(rootDir string) ([]ScannedSkill, error) {
	skillDirs, err := utils.FindSkillDirs(rootDir)
	if err != nil {
		return nil, err
	}

	var scanned []ScannedSkill
	var errorsCount int

	for _, dir := range skillDirs {
		frontmatter, body, err := ParseSkillMd(dir)
		if err != nil {
			if _, ok := err.(*utils.AxenError); ok {
				utils.Warn("Skipping invalid skill: %s", err.Error())
				errorsCount++
				continue
			}
			return nil, err
		}

		rel, _ := filepath.Rel(rootDir, dir)
		if rel == "." {
			rel = ""
		}

		scanned = append(scanned, ScannedSkill{
			Frontmatter:  frontmatter,
			DirPath:      dir,
			RelativePath: rel,
			Body:         body,
		})
	}

	if errorsCount > 0 {
		utils.Warn("%d skill(s) skipped due to invalid frontmatter", errorsCount)
	}

	return scanned, nil
}
