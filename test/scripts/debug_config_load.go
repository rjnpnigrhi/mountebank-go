package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mountebank-testing/mountebank-go/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_config_load <file>")
		os.Exit(1)
	}

	configFile := os.Args[1]
	fmt.Printf("Loading config from %s\n", configFile)

	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d imposters\n", len(cfg.Imposters))

	for i, imp := range cfg.Imposters {
		fmt.Printf("Imposter %d: Port=%d, Stubs=%d\n", i, imp.Port, len(imp.Stubs))
		if len(imp.Stubs) > 0 {
			// Print first stub details
			bytes, _ := json.MarshalIndent(imp.Stubs[0], "", "  ")
			fmt.Printf("First stub: %s\n", string(bytes))
		}
	}
}
