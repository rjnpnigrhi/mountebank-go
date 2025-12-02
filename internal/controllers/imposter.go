package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mountebank-testing/mountebank-go/internal/models"
	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// ImposterController handles single imposter endpoints
type ImposterController struct {
	repository *models.ImposterRepository
	logger     *util.Logger
}

// NewImposterController creates a new imposter controller
func NewImposterController(repository *models.ImposterRepository, logger *util.Logger) *ImposterController {
	return &ImposterController{
		repository: repository,
		logger:     logger,
	}
}

// Get handles GET /imposters/:id
func (ic *ImposterController) Get(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get query parameters
	replayable := r.URL.Query().Get("replayable") == "true"
	removeProxies := r.URL.Query().Get("removeProxies") == "true"

	options := map[string]interface{}{
		"replayable":    replayable,
		"requests":      true,
		"removeProxies": removeProxies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(options))
}

// Delete handles DELETE /imposters/:id
func (ic *ImposterController) Delete(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Delete(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get query parameters
	replayable := r.URL.Query().Get("replayable") == "true"
	removeProxies := r.URL.Query().Get("removeProxies") == "true"

	options := map[string]interface{}{
		"replayable":    replayable,
		"requests":      true,
		"removeProxies": removeProxies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(options))
}

// PutStubs handles PUT /imposters/:id/stubs
func (ic *ImposterController) PutStubs(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var request struct {
		Stubs []models.Stub `json:"stubs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Replace all stubs
	if err := imposter.Stubs().ReplaceAll(request.Stubs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
	}))
}

// PostStub handles POST /imposters/:id/stubs
func (ic *ImposterController) PostStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var request struct {
		Stub models.Stub `json:"stub"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stub := request.Stub

	// Get index from query parameter
	indexStr := r.URL.Query().Get("index")
	if indexStr != "" {
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			http.Error(w, "invalid index", http.StatusBadRequest)
			return
		}

		if err := imposter.Stubs().InsertAtIndex(stub, index); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		if err := imposter.Stubs().Add(stub); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
	}))
}

// DeleteStub handles DELETE /imposters/:id/stubs/:stubIndex
func (ic *ImposterController) DeleteStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	vars := mux.Vars(r)
	stubIndex, err := strconv.Atoi(vars["stubIndex"])
	if err != nil {
		http.Error(w, "invalid stub index", http.StatusBadRequest)
		return
	}

	if err := imposter.Stubs().DeleteAtIndex(stubIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
	}))
}

// PutStub handles PUT /imposters/:id/stubs/:stubIndex
func (ic *ImposterController) PutStub(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	vars := mux.Vars(r)
	stubIndex, err := strconv.Atoi(vars["stubIndex"])
	if err != nil {
		http.Error(w, "invalid stub index", http.StatusBadRequest)
		return
	}

	var request struct {
		Stub models.Stub `json:"stub"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := imposter.Stubs().ReplaceAtIndex(request.Stub, stubIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
	}))
}

// ResetRequests handles DELETE /imposters/:id/savedRequests
func (ic *ImposterController) ResetRequests(w http.ResponseWriter, r *http.Request) {
	port, err := ic.getPortFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imposter, err := ic.repository.Get(port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := imposter.ResetRequests(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imposter.ToJSON(map[string]interface{}{
		"replayable": true,
		"requests":   false,
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
