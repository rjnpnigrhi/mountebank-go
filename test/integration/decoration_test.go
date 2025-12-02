package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
)

func TestResponseDecoration(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2527,
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

	// Create an HTTP imposter with decorate behavior
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4548,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Original Body",
							"headers": map[string]interface{}{
								"X-Original": "true",
							},
						},
						"behaviors": []map[string]interface{}{
							{
								"decorate": `function(config) {
									config.response.statusCode = 201;
									config.response.body = "Decorated Body";
									config.response.headers['X-Decorated'] = 'true';
									// Return nothing to imply modification in place
								}`,
							},
						},
					},
				},
			},
		},
	}

	// Create imposter
	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2527/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test decoration
	testResp, err := http.Get("http://localhost:4548/")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	// Check status code
	if testResp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", testResp.StatusCode)
	}

	// Check body
	respBody, _ := io.ReadAll(testResp.Body)
	if string(respBody) != "Decorated Body" {
		t.Errorf("Expected body 'Decorated Body', got '%s'", string(respBody))
	}

	// Check headers
	if testResp.Header.Get("X-Decorated") != "true" {
		t.Errorf("Expected X-Decorated header to be 'true', got '%s'", testResp.Header.Get("X-Decorated"))
	}
	if testResp.Header.Get("X-Original") != "true" {
		t.Errorf("Expected X-Original header to be 'true', got '%s'", testResp.Header.Get("X-Original"))
	}
}
