package controllers

import (
	"encoding/json"
	"io"
	"net/http"

	"strings"

	"github.com/mountebank-testing/mountebank-go/internal/models"
	httpproto "github.com/mountebank-testing/mountebank-go/internal/protocols/http"
	httpsproto "github.com/mountebank-testing/mountebank-go/internal/protocols/https"
	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/mountebank-testing/mountebank-go/internal/web"
)

// ImpostersController handles imposter collection endpoints
type ImpostersController struct {
	repository *models.ImposterRepository
	renderer   *web.Renderer
	logger     *util.Logger
}

// NewImpostersController creates a new imposters controller
func NewImpostersController(repository *models.ImposterRepository, renderer *web.Renderer, logger *util.Logger) *ImpostersController {
	return &ImpostersController{
		repository: repository,
		renderer:   renderer,
		logger:     logger,
	}
}

// Get handles GET /imposters
func (ic *ImpostersController) Get(w http.ResponseWriter, r *http.Request) {
	imposters := ic.repository.GetAll()

	// Parse query parameters
	replayable := r.URL.Query().Get("replayable") == "true"
	removeProxies := r.URL.Query().Get("removeProxies") == "true"

	// Check if client accepts HTML (browser)
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		// Convert to simple JSON for template
		imposterList := make([]interface{}, 0, len(imposters))
		for _, imposter := range imposters {
			imposterList = append(imposterList, imposter.ToJSON(map[string]interface{}{
				"replayable":    replayable,
				"removeProxies": removeProxies,
				"requests":      false,
			}))
		}

		err := ic.renderer.Render(w, "imposters", map[string]interface{}{
			"imposters": imposterList,
		})
		if err != nil {
			ic.logger.Errorf("Failed to render imposters: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Convert to JSON format
	result := make(map[string]interface{})
	result["imposters"] = make([]interface{}, 0)

	imposterList := make([]interface{}, 0, len(imposters))
	for _, imposter := range imposters {
		imposterList = append(imposterList, imposter.ToJSON(map[string]interface{}{
			"replayable":    replayable,
			"removeProxies": removeProxies,
			"requests":      false,
		}))
	}

	result["imposters"] = imposterList

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Post handles POST /imposters
func (ic *ImpostersController) Post(w http.ResponseWriter, r *http.Request) {
	var config models.ImposterConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create imposter based on protocol
	imposter, err := ic.createImposter(&config)
	if err != nil {
		ic.logger.Errorf("Error creating imposter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add to repository
	if err := ic.repository.Add(imposter); err != nil {
		ic.logger.Errorf("Error adding imposter: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return imposter info
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
	}))
}

// Delete handles DELETE /imposters
func (ic *ImpostersController) Delete(w http.ResponseWriter, r *http.Request) {
	imposters, err := ic.repository.DeleteAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return deleted imposters
	result := make(map[string]interface{})
	imposterList := make([]interface{}, 0, len(imposters))
	for _, imposter := range imposters {
		imposterList = append(imposterList, imposter.ToJSON(map[string]interface{}{
			"replayable": true,
			"requests":   true,
		}))
	}

	result["imposters"] = imposterList

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Put handles PUT /imposters
func (ic *ImpostersController) Put(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var impostersConfig []models.ImposterConfig

	// Trim whitespace to check first character
	trimmedBody := strings.TrimSpace(string(body))
	if strings.HasPrefix(trimmedBody, "{") {
		// Wrapped object
		var wrappedRequest struct {
			Imposters []models.ImposterConfig `json:"imposters"`
		}
		if err := json.Unmarshal(body, &wrappedRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		impostersConfig = wrappedRequest.Imposters
	} else if strings.HasPrefix(trimmedBody, "[") {
		// Raw array
		if err := json.Unmarshal(body, &impostersConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Invalid JSON: must be an object or an array", http.StatusBadRequest)
		return
	}

	// Delete all existing imposters
	ic.repository.DeleteAll()

	// Create new imposters
	imposters := make([]*models.Imposter, 0, len(impostersConfig))
	for _, config := range impostersConfig {
		imposter, err := ic.createImposter(&config)
		if err != nil {
			ic.logger.Errorf("Error creating imposter: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := ic.repository.Add(imposter); err != nil {
			ic.logger.Errorf("Error adding imposter: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		imposters = append(imposters, imposter)
	}

	// Return imposters
	result := make(map[string]interface{})
	imposterList := make([]interface{}, 0, len(imposters))
	for _, imposter := range imposters {
		imposterList = append(imposterList, imposter.ToJSON(map[string]interface{}{
			"replayable": true,
			"requests":   false,
		}))
	}

	result["imposters"] = imposterList

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// createImposter creates an imposter based on protocol
func (ic *ImpostersController) createImposter(config *models.ImposterConfig) (*models.Imposter, error) {
	logger := ic.logger.WithScope(config.Protocol + ":" + string(rune(config.Port)))

	switch config.Protocol {
	case "http":
		return ic.createHTTPImposter(config, logger)

	case "https":
		return ic.createHTTPSImposter(config, logger)
	case "tcp":
		// TODO: Implement TCP
		ic.logger.Warn("TCP protocol not yet implemented")
		return nil, util.NewProtocolError("TCP protocol not yet implemented", config.Protocol, nil)
	case "smtp":
		// TODO: Implement SMTP
		ic.logger.Warn("SMTP protocol not yet implemented")
		return nil, util.NewProtocolError("SMTP protocol not yet implemented", config.Protocol, nil)
	default:
		return nil, util.NewProtocolError("unknown protocol", config.Protocol, nil)
	}
}

// createHTTPImposter creates an HTTP imposter
func (ic *ImpostersController) createHTTPImposter(config *models.ImposterConfig, logger *util.Logger) (*models.Imposter, error) {
	// Create a temporary imposter to get the response function
	var imposter *models.Imposter

	// Create HTTP server
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

// createHTTPSImposter creates an HTTPS imposter
func (ic *ImpostersController) createHTTPSImposter(config *models.ImposterConfig, logger *util.Logger) (*models.Imposter, error) {
	// Create a temporary imposter to get the response function
	var imposter *models.Imposter

	// Create HTTPS server
	server, err := httpsproto.Create(config, logger, func(request *models.Request, details map[string]interface{}) (*models.Response, error) {
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
