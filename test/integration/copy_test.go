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

func TestCopyBehavior(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2530,
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

	// Create imposter with copy behavior
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4551,
		"stubs": []map[string]interface{}{
			{
				"responses": []map[string]interface{}{
					{
						"is": map[string]interface{}{
							"statusCode": 200,
							"body":       "Hello ${NAME}, your ID is ${ID}",
							"headers": map[string]interface{}{
								"X-Request-ID": "${ID}",
							},
						},
						"behaviors": []map[string]interface{}{
							{
								"copy": []map[string]interface{}{
									{
										"from": "query.name",
										"into": "${NAME}",
									},
									{
										"from": "path",
										"into": "${ID}",
										"using": map[string]interface{}{
											"method": "regex",
											"selector": "/users/(\\d+)",
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
	resp, err := http.Post("http://localhost:2530/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test copy behavior
	testResp, err := http.Get("http://localhost:4551/users/123?name=Alice")
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp.Body.Close()

	// Check status code
	if testResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", testResp.StatusCode)
	}

	// Check body
	respBody, _ := io.ReadAll(testResp.Body)
	expectedBody := "Hello Alice, your ID is 123"
	if string(respBody) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(respBody))
	}

	// Check headers
	if testResp.Header.Get("X-Request-ID") != "123" {
		t.Errorf("Expected X-Request-ID header to be '123', got '%s'", testResp.Header.Get("X-Request-ID"))
	}
}
