package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mountebank-testing/mountebank-go/internal/models"
	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// Server represents an HTTP imposter server
type Server struct {
	port       int
	server     *http.Server
	listener   net.Listener
	logger     *util.Logger
	stubs      *models.StubRepository
	getResponse func(*models.Request, map[string]interface{}) (*models.Response, error)
	allowCORS  bool
}

// Create creates a new HTTP server
func Create(config *models.ImposterConfig, logger *util.Logger, getResponse func(*models.Request, map[string]interface{}) (*models.Response, error)) (*Server, error) {
	port := config.Port
	if port == 0 {
		// Find available port
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, err
		}
		port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	stubs := models.NewStubRepository(config.Stubs, logger)

	s := &Server{
		port:        port,
		logger:      logger,
		stubs:       stubs,
		getResponse: getResponse,
		allowCORS:   config.AllowCORS,
	}

	// Create HTTP handler
	handler := http.HandlerFunc(s.handleRequest)

	// Create HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	// Start listening
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return nil, err
	}
	s.listener = listener

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	logger.Infof("HTTP server started on port %d", port)

	return s, nil
}

// handleRequest handles incoming HTTP requests
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		msg := fmt.Sprintf("[IMPOSTER:%d] %s %s took %v", s.port, r.Method, r.URL.String(), duration)
		if duration > 100*time.Millisecond {
			msg += " (SLOW)"
		}
		s.logger.Info(msg)
	}()

	// Handle CORS
	if s.allowCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// Convert HTTP request to mountebank request
	request, err := s.httpToRequest(r)
	if err != nil {
		s.logger.Errorf("Error converting request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get response from imposter
	response, err := s.getResponse(request, nil)
	if err != nil {
		s.logger.Errorf("Error getting response: %v", err)
		if strings.Contains(err.Error(), "invalid injection") {
			// Return JSON error to match Mountebank behavior
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":    "invalid injection",
						"message": err.Error(),
					},
				},
			})
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Check if blocked
	if response.Blocked {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert mountebank response to HTTP response
	s.responseToHTTP(response, w)
}

// httpToRequest converts an HTTP request to a mountebank request
func (s *Server) httpToRequest(r *http.Request) (*models.Request, error) {
	// Read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Parse query parameters
	query := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			query[key] = values[0]
		} else {
			query[key] = values
		}
	}

	// Parse headers
	headers := make(map[string]interface{})
	for key, values := range r.Header {
		if len(values) == 1 {
			headers[key] = values[0]
		} else {
			headers[key] = values
		}
	}

	// Try to parse body as JSON
	var body interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			// If not JSON, use as string
			body = string(bodyBytes)
		}
	}

	return &models.Request{
		Protocol:  "http",
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     query,
		Headers:   headers,
		Body:      body,
		IP:        r.RemoteAddr,
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// responseToHTTP converts a mountebank response to an HTTP response
func (s *Server) responseToHTTP(response *models.Response, w http.ResponseWriter) {
	// Set status code
	statusCode := response.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}

	// Set headers
	if response.Headers != nil {
		for key, value := range response.Headers {
			switch v := value.(type) {
			case string:
				w.Header().Set(key, v)
			case []string:
				for _, val := range v {
					w.Header().Add(key, val)
				}
			default:
				w.Header().Set(key, fmt.Sprint(v))
			}
		}
	}

	// Write status code
	w.WriteHeader(statusCode)

	// Write body
	if response.Body != nil {
		switch body := response.Body.(type) {
		case string:
			w.Write([]byte(body))
		case []byte:
			w.Write(body)
		default:
			// Try to marshal as JSON
			if data, err := json.Marshal(body); err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			} else {
				w.Write([]byte(fmt.Sprint(body)))
			}
		}
	}
}

// Port returns the port the server is listening on
func (s *Server) Port() int {
	return s.port
}

// Stubs returns the stub repository
func (s *Server) Stubs() *models.StubRepository {
	return s.stubs
}

// Close closes the HTTP server
func (s *Server) Close(callback func()) error {
	if s.server != nil {
		if err := s.server.Close(); err != nil {
			s.logger.Errorf("Error closing HTTP server: %v", err)
		}
	}
	if callback != nil {
		callback()
	}
	return nil
}

// Metadata returns server metadata
func (s *Server) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"port": s.port,
	}
}

// Encoding returns the encoding used by the server
func (s *Server) Encoding() string {
	return "utf8"
}
