package core

import (
	"encoding/json"
	"os"

	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
)

func ReadConfig() (*models.Config, error) {
	configPath := resolvers.GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config := models.NewConfig()
			config.Targets = models.GetDefaultTargets()
			return config, nil
		}
		return nil, err
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		config := models.NewConfig()
		config.Targets = models.GetDefaultTargets()
		return config, nil
	}

	return &config, nil
}

func WriteConfig(config *models.Config) error {
	return utils.WriteJson(resolvers.GetConfigPath(), config)
}
