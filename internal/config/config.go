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

// processIncludes recursively processes EJS include tags (deprecated name, uses processTags)
func processIncludes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return processTags(path, data)
}

// processTags recursively processes EJS include and stringify tags
func processTags(path string, data []byte) ([]byte, error) {
	// 1. Process includes first (recursive)
	includeRe := regexp.MustCompile(`<%[-=]?\s*include\s*\(\s*["'](.+?)["']\s*\)\s*%>`)

	data = includeRe.ReplaceAllFunc(data, func(match []byte) []byte {
		matches := includeRe.FindSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		includePath := string(matches[1])
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(path), includePath)
		}

		includedData, err := os.ReadFile(includePath)
		if err != nil {
			fmt.Printf("Error reading included file %s: %v\n", includePath, err)
			return match
		}

		// Recursively process tags in the included file
		processedIncluded, err := processTags(includePath, includedData)
		if err != nil {
			fmt.Printf("Error processing tags in %s: %v\n", includePath, err)
			return match
		}
		return processedIncluded
	})

	// 2. Process stringify
	// Match <%- stringify(filename, 'path') %>
	// We ignore the 'filename' argument as we track path ourselves, but we need to parse the second argument
	stringifyRe := regexp.MustCompile(`<%[-=]?\s*stringify\s*\(\s*[^,]+,\s*["'](.+?)["']\s*\)\s*%>`)

	data = stringifyRe.ReplaceAllFunc(data, func(match []byte) []byte {
		matches := stringifyRe.FindSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		relPath := string(matches[1])
		absPath := relPath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(filepath.Dir(path), relPath)
		}

		fileData, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Printf("Error reading stringified file %s: %v\n", absPath, err)
			return match
		}

		// Convert to JSON string (escaped)
		// We use json.Marshal to get a quoted string "..."
		// But usually stringify is used inside quotes: "inject": "<%- stringify(...) %>"
		// If we return "...", we get "inject": ""..."" (double quotes).
		// Mountebank usage: "inject": "<%- stringify(...) %>"
		// So we need the content ESCAPED but WITHOUT checking surrounding quotes from me.
		// Wait. <%- stringify %> output is just the chars.
		// If I have "key": "VAL", and VAL comes from stringify.
		// If stringify returns `code "quote"`, result is "key": "code "quote"". INVALID JSON.
		// So stringify MUST escape quotes.
		// But if stringify returns `"code \"quote\""`, result is "key": ""code \"quote\""".
		// Usually stringify returns the _inner_ string content, escaped.

		marshaled, _ := json.Marshal(string(fileData))
		s := string(marshaled)
		// json.Marshal returns "content". We want content.
		// Strip outer quotes
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return []byte(s[1 : len(s)-1])
		}
		return []byte(s)
	})

	return data, nil
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
