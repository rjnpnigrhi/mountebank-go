package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONBodyIsStringInAPI(t *testing.T) {
	// Start mountebank server
	config := &server.Config{
		Port:           2530,
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

	mbURL := "http://localhost:2530"
	imposterPort := 4550

	imposterConfig := map[string]interface{}{
		"port":           imposterPort,
		"protocol":       "http",
		"recordRequests": true,
	}

	// 1. Create Imposter
	resp, err := http.Post(mbURL+"/imposters", "application/json", createBody(t, imposterConfig))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Wait for start
	time.Sleep(100 * time.Millisecond)

	// 2. Send JSON request
	jsonBody := `{"foo": "bar", "num": 123}`
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/", imposterPort), "application/json", strings.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 3. Get Imposter
	resp, err = http.Get(fmt.Sprintf("%s/imposters/%d", mbURL, imposterPort))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse generic map to inspect type
	var imposterDetail map[string]interface{}
	err = json.Unmarshal(bodyBytes, &imposterDetail)
	require.NoError(t, err)

	requests, ok := imposterDetail["requests"].([]interface{})
	require.True(t, ok, "requests should be a list")
	require.Len(t, requests, 1)

	req1 := requests[0].(map[string]interface{})
	body := req1["body"]

	// The fix should ensure this is a string
	_, isString := body.(string)
	assert.True(t, isString, "Expected body to be a string, got %T: %v", body, body)

	if isString {
		strBody := body.(string)
		assert.Contains(t, strBody, "foo")
		assert.Contains(t, strBody, "bar")
	}
}

func createBody(t *testing.T, v interface{}) io.Reader {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}
