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

// User provided struct
type ImposterRequest struct {
	RequestFrom string `json:"requestFrom"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       struct {
	} `json:"query"`
	Headers struct {
		Host           string `json:"Host"`
		UserAgent      string `json:"User-Agent"`
		ContentLength  string `json:"Content-Length"`
		Accept         string `json:"Accept"`
		Authorization  string `json:"Authorization"`
		ContentType    string `json:"Content-Type"`
		IdempotencyKey string `json:"Idempotency_key"`
		SpanID         string `json:"Span_id"`
		TraceID        string `json:"Trace_id"`
		AcceptEncoding string `json:"Accept-Encoding"`
	} `json:"headers"`

	Body      string    `json:"body"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type ImposterResponse struct {
	Requests []ImposterRequest `json:"requests"`
}

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

	// Verify we can unmarshal into user struct
	var imposterResp ImposterResponse
	err = json.Unmarshal(bodyBytes, &imposterResp)
	require.NoError(t, err, "Should be able to unmarshal API response into user struct")

	require.Len(t, imposterResp.Requests, 1)
	req := imposterResp.Requests[0]

	// Verify Body is a JSON string
	assert.Contains(t, req.Body, "foo")
	assert.Contains(t, req.Body, "bar")

	// Double check it's valid JSON itself
	var bodyJSON map[string]interface{}
	err = json.Unmarshal([]byte(req.Body), &bodyJSON)
	require.NoError(t, err, "Body string should be valid JSON")
	assert.Equal(t, "bar", bodyJSON["foo"])
}

func createBody(t *testing.T, v interface{}) io.Reader {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}
