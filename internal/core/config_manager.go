package core

import (
	"github.com/harishphk/axen/internal/models"
	"github.com/harishphk/axen/internal/resolvers"
	"github.com/harishphk/axen/internal/utils"
)

func InitConfig() (*models.Config, error) {
	configPath := resolvers.GetConfigPath()

	if !utils.PathExists(configPath) {
		config := &models.Config{Targets: models.GetDefaultTargets()}
		err := WriteConfig(config)
		return config, err
	}

	return ReadConfig()
}
