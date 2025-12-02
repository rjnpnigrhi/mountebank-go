package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mountebank-testing/mountebank-go/internal/controllers"
	"github.com/mountebank-testing/mountebank-go/internal/models"
	httpproto "github.com/mountebank-testing/mountebank-go/internal/protocols/http"
	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/mountebank-testing/mountebank-go/internal/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

// Config represents server configuration
type Config struct {
	Port          int
	Host          string
	LogLevel      string
	AllowInjection bool
	IPWhitelist   []string
	APIKey        string
}

// Server represents the mountebank server
type Server struct {
	config     *Config
	httpServer *http.Server
	logger     *util.Logger
	repository *models.ImposterRepository
	renderer   *web.Renderer
}

// New creates a new mountebank server
func New(config *Config) (*Server, error) {
	logger := util.NewLogger(config.LogLevel)

	repository := models.NewImposterRepository(logger)

	// Initialize renderer
	viewsFS, err := fs.Sub(web.GetAssets(), "views")
	if err != nil {
		return nil, err
	}
	renderer, err := web.NewRenderer(viewsFS)
	if err != nil {
		return nil, err
	}

	s := &Server{
		config:     config,
		logger:     logger,
		repository: repository,
		renderer:   renderer,
	}

	// Create router
	router := s.createRouter()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s, nil
}

// createRouter creates the HTTP router
func (s *Server) createRouter() http.Handler {
	router := mux.NewRouter()

	// Create controllers
	impostersController := controllers.NewImpostersController(s.repository, s.renderer, s.logger)
	imposterController := controllers.NewImposterController(s.repository, s.logger)

	// Routes
	router.HandleFunc("/", s.handleHome).Methods("GET")
	router.HandleFunc("/imposters", impostersController.Get).Methods("GET")
	router.HandleFunc("/imposters", impostersController.Post).Methods("POST")
	router.HandleFunc("/imposters", impostersController.Delete).Methods("DELETE")
	router.HandleFunc("/imposters", impostersController.Put).Methods("PUT")
	
	router.HandleFunc("/imposters/{id}", imposterController.Get).Methods("GET")
	router.HandleFunc("/imposters/{id}", imposterController.Delete).Methods("DELETE")
	router.HandleFunc("/imposters/{id}/stubs", imposterController.PutStubs).Methods("PUT")
	router.HandleFunc("/imposters/{id}/stubs", imposterController.PostStub).Methods("POST")
	router.HandleFunc("/imposters/{id}/stubs/{stubIndex}", imposterController.PutStub).Methods("PUT")
	router.HandleFunc("/imposters/{id}/stubs/{stubIndex}", imposterController.DeleteStub).Methods("DELETE")
	router.HandleFunc("/imposters/{id}/savedRequests", imposterController.ResetRequests).Methods("DELETE")
	router.HandleFunc("/imposters/{id}/savedProxyResponses", imposterController.DeleteSavedProxyResponses).Methods("DELETE")
	
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	router.HandleFunc("/config", s.handleConfig).Methods("GET")

	// Static assets
	publicFS, _ := fs.Sub(web.GetAssets(), "public")
	fileServer := http.FileServer(http.FS(publicFS))
	router.PathPrefix("/images/").Handler(fileServer)
	router.PathPrefix("/scripts/").Handler(fileServer)
	router.PathPrefix("/stylesheets/").Handler(fileServer)
	router.Handle("/favicon.ico", fileServer)

	// Add CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Catch-all for static pages (must be last)
	router.PathPrefix("/").HandlerFunc(s.handlePage).Methods("GET")

	return corsHandler.Handler(router)
}

// handlePage handles static pages
func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	// Only handle HTML requests
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path
	if path == "/" {
		s.handleHome(w, r)
		return
	}

	// Remove leading slash
	name := strings.TrimPrefix(path, "/")
	
	// Render template
	err := s.renderer.Render(w, name, map[string]interface{}{
		"port": s.config.Port,
		"version": "2.9.3-go",
		"notices": []interface{}{}, // TODO: notices
	})
	if err != nil {
		// If template not found, 404
		s.logger.Debugf("Template not found for %s: %v", name, err)
		http.NotFound(w, r)
	}
}

// handleHome handles the home page
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Check if client accepts HTML (browser)
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		err := s.renderer.Render(w, "index", map[string]interface{}{
			"port": s.config.Port,
			"version": "2.9.3-go",
			"notices": []interface{}{}, // TODO: notices
		})
		if err != nil {
			s.logger.Errorf("Failed to render index: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"_links": map[string]interface{}{
			"imposters": map[string]string{
				"href": "/imposters",
			},
			"config": map[string]string{
				"href": "/config",
			},
			"metrics": map[string]string{
				"href": "/metrics",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Simple JSON response
	json.NewEncoder(w).Encode(response)
}

// handleConfig handles the config endpoint
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"version": "2.9.3-go",
		"options": map[string]interface{}{
			"port":           s.config.Port,
			"host":           s.config.Host,
			"allowInjection": s.config.AllowInjection,
			"ipWhitelist":    s.config.IPWhitelist,
		},
		"process": map[string]interface{}{
			"platform": "go",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(config)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Infof("mountebank-go now taking orders - point your browser to http://%s:%d/ for help", s.config.Host, s.config.Port)

	if s.config.AllowInjection {
		s.logger.Warnf("Running with --allowInjection set. See http://%s:%d/docs/security for security info", s.config.Host, s.config.Port)
	}

	return s.httpServer.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	s.logger.Info("Shutting down server...")

	// Stop all imposters
	s.repository.StopAll()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	s.logger.Info("Adios - see you soon?")
	return nil
}

// Repository returns the imposter repository
func (s *Server) Repository() *models.ImposterRepository {
	return s.repository
}

// CreateImposter creates and adds an imposter to the server
func (s *Server) CreateImposter(config *models.ImposterConfig) error {
	// Create logger for this imposter
	logger := s.logger.WithScope(fmt.Sprintf("%s:%d", config.Protocol, config.Port))

	var imposter *models.Imposter
	var err error

	switch config.Protocol {
	case "http":
		imposter, err = s.createHTTPImposter(config, logger)
	case "https":
		s.logger.Warn("HTTPS protocol not yet implemented")
		imposter, err = s.createHTTPImposter(config, logger)
	case "tcp":
		return util.NewProtocolError("TCP protocol not yet implemented", config.Protocol, nil)
	case "smtp":
		return util.NewProtocolError("SMTP protocol not yet implemented", config.Protocol, nil)
	default:
		return util.NewProtocolError("unknown protocol", config.Protocol, nil)
	}

	if err != nil {
		return err
	}

	return s.repository.Add(imposter)
}

// createHTTPImposter creates an HTTP imposter
func (s *Server) createHTTPImposter(config *models.ImposterConfig, logger *util.Logger) (*models.Imposter, error) {
	// Create a temporary imposter to get the response function
	var imposter *models.Imposter

	// Create HTTP server
	// We need to import httpproto
	server, err := httpproto.Create(config, logger, func(request *models.Request, details map[string]interface{}) (*models.Response, error) {
		return imposter.GetResponseFor(request, details)
	})
	if err != nil {
		return nil, err
	}

	// Create imposter with the server's close function
	imposter = models.NewImposter(config, logger, server.Close)
	
	// Update port if it was auto-assigned
	if config.Port == 0 {
		config.Port = server.Port()
	}

	return imposter, nil
}
