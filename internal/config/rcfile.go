package config

import (
	"bufio"
	"os"
	"strings"
)

// ParseRCFile parses a run commands file
func ParseRCFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by first space
		parts := strings.SplitN(line, " ", 2)
		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		} else {
			// Boolean flags might not have a value, assume "true" if it's a flag?
			// Or maybe the format requires value?
			// Mountebank docs say: "arguments are space-separated"
			// e.g. "port 3535"
			// For boolean flags like "--mock", it might be "mock" or "mock true"?
			// Let's assume value is required or empty string implies boolean true if mapped later.
			// But for now, empty string.
			value = "true" // Default to true for boolean flags without value
		}

		// Remove leading dashes if present (though rcfile usually doesn't have them)
		key = strings.TrimPrefix(key, "--")
		
		config[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}
