package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mountebank-testing/mountebank-go/internal/models"
	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/mountebank-testing/mountebank-go/internal/web"
)

// ImposterController handles single imposter endpoints
type ImposterController struct {
	repository *models.ImposterRepository
	logger     *util.Logger
	renderer   *web.Renderer
}

// NewImposterController creates a new imposter controller
func NewImposterController(repository *models.ImposterRepository, logger *util.Logger, renderer *web.Renderer) *ImposterController {
	return &ImposterController{
		repository: repository,
		logger:     logger,
		renderer:   renderer,
	}
}

// Get handles GET /imposters/:id
func (ic *ImposterController) Get(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	// Get query parameters
	replayable := r.URL.Query().Get("replayable") == "true"
	removeProxies := r.URL.Query().Get("removeProxies") == "true"

	options := map[string]interface{}{
		"replayable":    replayable,
		"requests":      true,
		"removeProxies": removeProxies,
		"stubs":         true,
	}

	// Check if client accepts HTML (browser)
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		info := imposter.ToJSON(options)

		// Convert struct to map for template access
		var imposterMap map[string]interface{}
		data, _ := json.Marshal(info)
		json.Unmarshal(data, &imposterMap)

		err := ic.renderer.Render(w, "imposter", imposterMap)
		if err != nil {
			ic.logger.Errorf("Failed to render imposter: %v", err)
			util.WriteError(w, err, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(options))
}

// Delete handles DELETE /imposters/:id
func (ic *ImposterController) Delete(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Delete(port)
	if err != nil {
		// Node.js returns {} and 200 OK if imposter not found
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	// Get query parameters
	replayable := r.URL.Query().Get("replayable") == "true"
	removeProxies := r.URL.Query().Get("removeProxies") == "true"

	options := map[string]interface{}{
		"replayable":    replayable,
		"requests":      true,
		"removeProxies": removeProxies,
		"stubs":         true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(options))
}

// PutStubs handles PUT /imposters/:id/stubs
func (ic *ImposterController) PutStubs(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	var request struct {
		Stubs []models.Stub `json:"stubs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		util.WriteError(w, util.NewInvalidJSONError(err.Error()), http.StatusBadRequest)
		return
	}

	// Replace all stubs
	if err := imposter.Stubs().ReplaceAll(request.Stubs); err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// PostStub handles POST /imposters/:id/stubs
func (ic *ImposterController) PostStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	var request struct {
		Stub models.Stub `json:"stub"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		util.WriteError(w, util.NewInvalidJSONError(err.Error()), http.StatusBadRequest)
		return
	}
	stub := request.Stub

	// Get index from query parameter
	indexStr := r.URL.Query().Get("index")
	if indexStr != "" {
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			util.WriteError(w, util.NewValidationError("invalid index", indexStr), http.StatusBadRequest)
			return
		}

		if err := imposter.Stubs().InsertAtIndex(stub, index); err != nil {
			util.WriteError(w, err, http.StatusBadRequest)
			return
		}
	} else {
		if err := imposter.Stubs().Add(stub); err != nil {
			util.WriteError(w, err, http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// DeleteStub handles DELETE /imposters/:id/stubs/:stubIndex
func (ic *ImposterController) DeleteStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	vars := mux.Vars(r)
	stubIndex, err := strconv.Atoi(vars["stubIndex"])
	if err != nil {
		util.WriteError(w, util.NewValidationError("invalid stub index", vars["stubIndex"]), http.StatusBadRequest)
		return
	}

	if err := imposter.Stubs().DeleteAtIndex(stubIndex); err != nil {
		util.WriteError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// PutStub handles PUT /imposters/:id/stubs/:stubIndex
func (ic *ImposterController) PutStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	vars := mux.Vars(r)
	stubIndex, err := strconv.Atoi(vars["stubIndex"])
	if err != nil {
		util.WriteError(w, util.NewValidationError("invalid stub index", vars["stubIndex"]), http.StatusBadRequest)
		return
	}

	var stub models.Stub
	if err := json.NewDecoder(r.Body).Decode(&stub); err != nil {
		util.WriteError(w, util.NewInvalidJSONError(err.Error()), http.StatusBadRequest)
		return
	}

	if err := imposter.Stubs().ReplaceAtIndex(stub, stubIndex); err != nil {
		util.WriteError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// ResetRequests handles DELETE /imposters/:id/savedRequests
func (ic *ImposterController) ResetRequests(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	if err := imposter.ResetRequests(); err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// DeleteSavedProxyResponses handles DELETE /imposters/:id/savedProxyResponses
func (ic *ImposterController) DeleteSavedProxyResponses(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		util.WriteError(w, util.NewMissingResourceError(err.Error(), port), http.StatusNotFound)
		return
	}

	if err := imposter.DeleteSavedProxyResponses(); err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"requests": true,
		"stubs":    true,
	}))
}

// getPortFromRequest extracts the port from the request
func (ic *ImposterController) getPortFromRequest(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	portStr := vars["id"]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, util.NewValidationError("invalid port", portStr)
	}

	return port, nil
}
