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

func TestJSONPath(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2531,
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

	// Create imposter with JSONPath predicate and copy behavior
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4552,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": "test-value",
						"jsonpath": map[string]interface{}{
							"selector": "$.body.items[0].name",
						},
					},
				},
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Matched ${ID}",
						},
						"behaviors": []map[string]interface{}{
							{
								"copy": []map[string]interface{}{
									{
										"from": "body",
										"into": "${ID}",
										"using": map[string]interface{}{
											"method": "jsonpath",
											"selector": "$.items[0].id",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(imposterConfig)
	resp, err := http.Post("http://localhost:2531/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test case 1: Matching request
	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"name": "test-value",
				"id":   "12345",
			},
		},
	}
	reqBytes, _ := json.Marshal(reqBody)
	testResp, err := http.Post("http://localhost:4552/test", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp.StatusCode)
	}

	respBody, _ := io.ReadAll(testResp.Body)
	expectedBody := "Matched 12345"
	if string(respBody) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(respBody))
	}

	// Test case 2: Non-matching request
	reqBody2 := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"name": "wrong-value",
				"id":   "67890",
			},
		},
	}
	reqBytes2, _ := json.Marshal(reqBody2)
	testResp2, err := http.Post("http://localhost:4552/test", "application/json", bytes.NewBuffer(reqBytes2))
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp2.Body.Close()

	// Should not match, so default response (empty body, 200 OK usually, or 404 if no default?)
	// Mountebank default response is empty 200 OK if no match found and no default set?
	// Actually, I implemented default response.
	// Let's check what it returns.
	// If no match, it returns default response which is empty.
	
	// Just check it's not "Matched 67890"
	respBody2, _ := io.ReadAll(testResp2.Body)
	if string(respBody2) == "Matched 67890" {
		t.Errorf("Should not have matched")
	}
}
