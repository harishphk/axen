package core

import (
	"axen/internal/models"
	"axen/internal/resolvers"
	"axen/internal/utils"
)

func ReadConfig() (*models.Config, error) {
	configPath := resolvers.GetConfigPath()

	if !utils.PathExists(configPath) {
		return &models.Config{Targets: models.GetDefaultTargets()}, nil
	}

	config, err := utils.ReadJson[models.Config](configPath)
	if err != nil {
		utils.Warn("Failed to parse axen-config.json, using defaults")
		return &models.Config{Targets: models.GetDefaultTargets()}, nil
	}

	return &config, nil
}

func WriteConfig(config *models.Config) error {
	if err := utils.EnsureDir(resolvers.GetAxenDir()); err != nil {
		return err
	}
	return utils.WriteJson(resolvers.GetConfigPath(), config)
}

func InitConfig() (*models.Config, error) {
	configPath := resolvers.GetConfigPath()

	if !utils.PathExists(configPath) {
		config := &models.Config{Targets: models.GetDefaultTargets()}
		err := WriteConfig(config)
		return config, err
	}

	return ReadConfig()
}
