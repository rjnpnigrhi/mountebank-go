package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mountebank-testing/mountebank-go/internal/models"
)

// Config represents the structure of the configuration file
type Config struct {
	Imposters []models.ImposterConfig `json:"imposters"`
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := processIncludes(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// processIncludes recursively processes EJS include tags
func processIncludes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Regex to match <%- include("filename") %> or <%- include('filename') %>
	// Handles optional whitespace and optional hyphen
	re := regexp.MustCompile(`<%[-=]?\s*include\s*\(\s*["'](.+?)["']\s*\)\s*%>`)

	processed := re.ReplaceAllFunc(data, func(match []byte) []byte {
		matches := re.FindSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		includePath := string(matches[1])
		
		// Resolve relative path
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(path), includePath)
		}

		includedData, err := processIncludes(includePath)
		if err != nil {
			// If we can't read the included file, return the original match
			// or maybe we should error out? For now, let's log/print and return match
			fmt.Printf("Error processing include %s: %v\n", includePath, err)
			return match
		}

		return includedData
	})

	return processed, nil
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
