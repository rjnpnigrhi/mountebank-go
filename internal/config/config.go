package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mountebank-testing/mountebank-go/internal/models"
)

// Config represents the structure of the configuration file
type Config struct {
	Imposters []models.ImposterConfig `json:"imposters"`
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// Save saves configuration to a file
func Save(path string, imposters []models.ImposterConfig) error {
	config := Config{
		Imposters: imposters,
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}

	return nil
}
