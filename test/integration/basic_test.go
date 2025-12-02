package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
)

func TestBasicHTTPImposter(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2525,
		Host:           "localhost",
		LogLevel:       "error",
		AllowInjection: false,
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

	// Create an HTTP imposter
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4545,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"headers": map[string]string{
								"Content-Type": "application/json",
							},
							"body": `{"message": "Hello from mountebank-go!"}`,
						},
					},
				},
			},
		},
	}

	// Create imposter
	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2525/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Wait for imposter to start
	time.Sleep(100 * time.Millisecond)

	// Test the imposter
	testResp, err := http.Get("http://localhost:4545/test")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp.StatusCode)
	}

	// Verify response body
	var responseBody map[string]interface{}
	if err := json.NewDecoder(testResp.Body).Decode(&responseBody); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if responseBody["message"] != "Hello from mountebank-go!" {
		t.Errorf("Expected message 'Hello from mountebank-go!', got %v", responseBody["message"])
	}

	// Clean up - delete imposter
	req, _ := http.NewRequest("DELETE", "http://localhost:2525/imposters/4545", nil)
	deleteResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete imposter: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 when deleting, got %d", deleteResp.StatusCode)
	}
}

func TestPredicateMatching(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2526,
		Host:           "localhost",
		LogLevel:       "error",
		AllowInjection: false,
		IPWhitelist:    []string{"*"},
	}

	srv, err := server.New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)
	defer srv.Stop()

	// Create imposter with predicates
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4546,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": map[string]interface{}{
							"path": "/test",
						},
					},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Matched!",
						},
					},
				},
			},
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 404,
							"body":       "Not found",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2526/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	// Test matching path
	testResp, _ := http.Get("http://localhost:4546/test")
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200 for matching path, got %d", testResp.StatusCode)
	}

	// Test non-matching path
	testResp2, _ := http.Get("http://localhost:4546/other")
	defer testResp2.Body.Close()

	if testResp2.StatusCode != 404 {
		t.Errorf("Expected status 404 for non-matching path, got %d", testResp2.StatusCode)
	}
}
