package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
)

func TestCORS(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2533,
		Host:           "localhost",
		LogLevel:       "info",
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

	// Create imposter with allowCORS=true
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4554,
		"allowCORS": true,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Hello CORS",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2533/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test case 1: Preflight OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "http://localhost:4554/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	
	client := &http.Client{}
	testResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200 for OPTIONS, got %d", testResp.StatusCode)
	}

	// Check CORS headers
	if testResp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", testResp.Header.Get("Access-Control-Allow-Origin"))
	}
	if testResp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("Expected Access-Control-Allow-Methods to be set")
	}

	// Test case 2: Normal request with Origin
	req2, _ := http.NewRequest("GET", "http://localhost:4554/test", nil)
	req2.Header.Set("Origin", "http://example.com")
	
	testResp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp2.Body.Close()

	if testResp2.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp2.StatusCode)
	}

	// Check CORS headers on normal response
	if testResp2.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", testResp2.Header.Get("Access-Control-Allow-Origin"))
	}
}
