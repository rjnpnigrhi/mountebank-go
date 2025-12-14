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

func TestXPath(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2532,
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

	// Create imposter with XPath predicate and copy behavior
	imposterConfig := map[string]interface{}{
		"protocol": "http",
		"port":     4553,
		"stubs": []map[string]interface{}{
			{
				"predicates": []map[string]interface{}{
					{
						"equals": "test-value",
						"xpath": map[string]interface{}{
							"selector": "//item/name",
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
											"method": "xpath",
											"selector": "//item/id",
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
	resp, err := http.Post("http://localhost:2532/imposters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create imposter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test case 1: Matching request
	reqBody := `<root>
		<item>
			<name>test-value</name>
			<id>12345</id>
		</item>
	</root>`
	
	testResp, err := http.Post("http://localhost:4553/test", "application/xml", bytes.NewBufferString(reqBody))
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
	reqBody2 := `<root>
		<item>
			<name>wrong-value</name>
			<id>67890</id>
		</item>
	</root>`
	
	testResp2, err := http.Post("http://localhost:4553/test", "application/xml", bytes.NewBufferString(reqBody2))
	if err != nil {
		t.Fatalf("Failed to call imposter: %v", err)
	}
	defer testResp2.Body.Close()

	respBody2, _ := io.ReadAll(testResp2.Body)
	if string(respBody2) == "Matched 67890" {
		t.Errorf("Should not have matched")
	}
}
