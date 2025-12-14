package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
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
	Port           int
	Host           string
	LogLevel       string
	AllowInjection bool
	IPWhitelist    []string
	APIKey         string
	LogFile        string
	NoLogFile      bool
	Datadir        string
	Origin         []string
	Debug          bool
	LocalOnly      bool
	ProtoFile      string
	Formatter      string
	NoParse        bool
	LogConfig      string
	ImpostersRepo  string
}

// Server represents the mountebank server
type Server struct {
	config     *Config
	httpServer *http.Server
	logger     *util.Logger
	repository *models.ImposterRepository
	renderer   *web.Renderer
}

var startTime = time.Now()

// New creates a new mountebank server
func New(config *Config) (*Server, error) {
	logger := util.NewLogger(config.LogLevel, config.LogFile, config.NoLogFile)

	// Initialize data store
	var dataStore models.DataStore
	var err error

	if config.ImpostersRepo != "" {
		dataStore, err = models.NewGojaDataStore(config.ImpostersRepo, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize custom repository: %v", err)
		}
		logger.Infof("Using custom imposters repository: %s", config.ImpostersRepo)
	} else if config.Datadir != "" {
		dataStore = models.NewFileSystemDataStore(config.Datadir, logger)
		logger.Infof("Using filesystem data store: %s", config.Datadir)
	} else {
		dataStore = &models.NoOpDataStore{}
	}

	repository := models.NewImposterRepository(logger, dataStore)

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

	// Load imposters from data store
	configs, err := dataStore.Load()
	if err != nil {
		s.logger.Errorf("Failed to load imposters from data store: %v", err)
	} else {
		for _, cfg := range configs {
			if err := s.CreateImposter(cfg); err != nil {
				s.logger.Errorf("Failed to create imposter from config on port %d: %v", cfg.Port, err)
			}
		}
	}

	return s, nil
}

// createRouter creates the HTTP router
func (s *Server) createRouter() http.Handler {
	router := mux.NewRouter()

	// Create controllers
	impostersController := controllers.NewImpostersController(s.repository, s.renderer, s.logger, s.config.AllowInjection, s.config.Debug)
	imposterController := controllers.NewImposterController(s.repository, s.logger, s.renderer)
	logsController := controllers.NewLogsController(s.logger, s.renderer)

	// Routes
	router.HandleFunc("/", s.handleHome).Methods("GET")
	router.HandleFunc("/feed", s.handleFeed).Methods("GET")
	router.HandleFunc("/faqs", s.handleStaticView("faqs", "FAQs")).Methods("GET")
	router.HandleFunc("/support", s.handleStaticView("support", "Support")).Methods("GET")
	router.PathPrefix("/docs/").HandlerFunc(s.handleDocs)

	router.HandleFunc("/imposters", impostersController.Get).Methods("GET")

	// Add middleware
	router.Use(s.loggingMiddleware)
	router.Use(s.ipWhitelistMiddleware)
	router.Use(s.apiKeyMiddleware)

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
	router.HandleFunc("/imposters/{id}/_requests", imposterController.PostRequest).Methods("POST")
	// router.HandleFunc("/imposters/{id}/_requests/{proxyResolutionKey}", imposterController.PostProxyResponse).Methods("POST")
	router.HandleFunc("/logs", logsController.Get).Methods("GET")

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
	allowedOrigins := s.config.Origin
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
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
		"port":    s.config.Port,
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
			"port":    s.config.Port,
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
			"logs": map[string]string{
				"href": "/logs",
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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cwd, _ := os.Getwd()

	config := map[string]interface{}{
		"version": "2.9.3-go",
		"options": map[string]interface{}{
			"port":           s.config.Port,
			"host":           s.config.Host,
			"logLevel":       s.config.LogLevel,
			"allowInjection": s.config.AllowInjection,
			"ipWhitelist":    s.config.IPWhitelist,
		},
		"process": map[string]interface{}{
			"nodeVersion":  runtime.Version(), // Using Go version as nodeVersion for template compatibility
			"architecture": runtime.GOARCH,
			"platform":     runtime.GOOS,
			"rss":          m.Sys,
			"heapTotal":    m.HeapSys,
			"heapUsed":     m.HeapAlloc,
			"uptime":       time.Since(startTime).Seconds(),
			"cwd":          cwd,
		},
	}

	// Check if client accepts HTML (browser)
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		err := s.renderer.Render(w, "config", config)
		if err != nil {
			s.logger.Errorf("Failed to render config: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(config)
}

// handleFeed handles the feed endpoint
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		host = "localhost:" + strconv.Itoa(s.config.Port)
	}

	// Mock release data for now
	releases := []map[string]interface{}{
		{
			"version": "2.9.3-go",
			"date":    time.Now().Format("2006-01-02"),
			"view":    "<p>Mountebank Go Port Release</p>",
		},
	}

	data := map[string]interface{}{
		"host":        host,
		"hasNextPage": false,
		"nextLink":    "",
		"releases":    releases,
	}

	w.Header().Set("Content-Type", "application/atom+xml")
	err := s.renderer.Render(w, "feed", data)
	if err != nil {
		s.logger.Errorf("Failed to render feed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleStaticView returns a handler for a static template
func (s *Server) handleStaticView(templateName, title string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.renderer.Render(w, templateName, map[string]interface{}{
			"title": title,
		})
		if err != nil {
			s.logger.Errorf("Failed to render %s: %v", templateName, err)
			http.NotFound(w, r)
		}
	}
}

// handleDocs handles documentation pages
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	// Remove .html if present (though links usually don't have it)
	path = strings.TrimSuffix(path, ".html")

	// Title is usually the last part of the path, capitalized
	parts := strings.Split(path, "/")
	title := parts[len(parts)-1]

	err := s.renderer.Render(w, path, map[string]interface{}{
		"title": title,
	})
	if err != nil {
		s.logger.Errorf("Failed to render docs %s: %v", path, err)
		http.NotFound(w, r)
	}
}

// loggingMiddleware logs request duration
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		// Filter out UI pages and static assets
		if strings.Contains(r.Header.Get("Accept"), "text/html") ||
			strings.HasPrefix(r.URL.Path, "/images/") ||
			strings.HasPrefix(r.URL.Path, "/scripts/") ||
			strings.HasPrefix(r.URL.Path, "/stylesheets/") ||
			r.URL.Path == "/favicon.ico" {
			return
		}

		msg := fmt.Sprintf("[ADMIN] %s %s took %v", r.Method, r.URL.String(), duration)
		if duration > 100*time.Millisecond {
			msg += " (SLOW)"
		}
		s.logger.Info(msg)
	})
}

// ipWhitelistMiddleware checks if the request IP is whitelisted
func (s *Server) ipWhitelistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if whitelist contains *
		for _, ip := range s.config.IPWhitelist {
			if ip == "*" {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Get IP from RemoteAddr
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Fallback if no port
			host = r.RemoteAddr
		}

		// Check if IP is in whitelist
		allowed := false
		for _, ip := range s.config.IPWhitelist {
			if ip == host {
				allowed = true
				break
			}
		}

		if !allowed {
			s.logger.Warnf("Blocked request from non-whitelisted IP: %s", host)
			http.Error(w, "You are not authorized to access this resource", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware checks for the correct API key
func (s *Server) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.APIKey != "" {
			key := r.Header.Get("x-api-key")
			if key != s.config.APIKey {
				s.logger.Warn("Blocked request with invalid or missing API key")
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
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

	// Define save function
	saveFunc := func(imp *models.Imposter) error {
		return s.repository.Save(imp)
	}

	switch config.Protocol {
	case "http":
		imposter, err = s.createHTTPImposter(config, logger, saveFunc)
	case "https":
		s.logger.Warn("HTTPS protocol not yet implemented")
		imposter, err = s.createHTTPImposter(config, logger, saveFunc)
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
func (s *Server) createHTTPImposter(config *models.ImposterConfig, logger *util.Logger, saveFunc func(*models.Imposter) error) (*models.Imposter, error) {
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
	imposter = models.NewImposter(config, logger, s.config.AllowInjection, server.Close, saveFunc)

	// Update port if it was auto-assigned
	if config.Port == 0 {
		config.Port = server.Port()
	}

	return imposter, nil
}
