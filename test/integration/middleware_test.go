package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
)

func TestMiddleware(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2528,
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

	// Test 1: Request Modification
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4549,
		"middleware": `function(config) {
			config.request.path = '/modified';
			config.request.headers['X-Modified'] = 'true';
		}`,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"path": "/modified",
							"headers": map[string]interface{}{
								"X-Modified": "true",
							},
						},
					},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Modified Match",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2528/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Call with original path
	testResp, err := http.Get("http://localhost:4549/original")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp.StatusCode)
	}

	// Test 2: Short-circuit Response
	imposterConfig2 := map[string]interface{}{
		"protocol": "http",
		"port":     4550,
		"middleware": `function(config) {
			if (config.request.path === '/blocked') {
				return {
					statusCode: 403,
					body: "Blocked by middleware"
				};
			}
		}`,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Allowed",
						},
					},
				},
			},
		},
	}

	body2, _ := json.Marshal(imposterConfig2)
	resp2, err := http.Post("http://localhost:2528/imposters", "application/json", bytes.NewBuffer(body2))
	if err != nil {
		t.Fatalf("Failed to create imposter 2: %v", err)
	}
	defer resp2.Body.Close()

	// Call blocked path
	testResp2, err := http.Get("http://localhost:4550/blocked")
	if err != nil {
		t.Fatalf("Failed to call imposter 2: %v", err)
	}
	defer testResp2.Body.Close()

	if testResp2.StatusCode != 403 {
		t.Errorf("Expected status 403, got %d", testResp2.StatusCode)
	}

	// Call allowed path
	testResp3, err := http.Get("http://localhost:4550/allowed")
	if err != nil {
		t.Fatalf("Failed to call imposter 2: %v", err)
	}
	defer testResp3.Body.Close()

	if testResp3.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp3.StatusCode)
	}
}
