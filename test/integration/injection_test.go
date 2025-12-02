package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
)

func TestPredicateInjection(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2526,
		Host:           "localhost",
		LogLevel:       "info",
		AllowInjection: true, // Enable injection
		IPWhitelist:    []string{"*"},
	}

	srv, err := server.New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	// Create an HTTP imposter with inject predicate
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4547,
		"defaultResponse": map[string]interface{}{
			"statusCode": 404,
		},
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"inject": "function(config, logger) { logger.info('Checking path: ' + config.request.path); return config.request.path === '/injected'; }",
					},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Injected Match!",
						},
					},
				},
			},
		},
	}

	// Create imposter
	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2526/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test matching path
	testResp, err := http.Get("http://localhost:4547/injected")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200 for matching path, got %d", testResp.StatusCode)
	}

	// Test non-matching path
	testResp2, err := http.Get("http://localhost:4547/other")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp2.Body.Close()

	if testResp2.StatusCode != 404 { // Should be 404 as no stub matches
		t.Errorf("Expected status 404 for non-matching path, got %d", testResp2.StatusCode)
	}
}
