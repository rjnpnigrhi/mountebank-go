package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"strings"

	"github.com/mountebank-testing/mountebank-go/internal/util"
	"github.com/mountebank-testing/mountebank-go/internal/web"
)

// LogsController handles logs endpoints
type LogsController struct {
	logger   *util.Logger
	renderer *web.Renderer
}

// NewLogsController creates a new logs controller
func NewLogsController(logger *util.Logger, renderer *web.Renderer) *LogsController {
	return &LogsController{
		logger:   logger,
		renderer: renderer,
	}
}

// Get handles GET /logs
func (lc *LogsController) Get(w http.ResponseWriter, r *http.Request) {
	startIndexStr := r.URL.Query().Get("startIndex")
	endIndexStr := r.URL.Query().Get("endIndex")

	startIndex := 0
	endIndex := -1 // All

	if startIndexStr != "" {
		if val, err := strconv.Atoi(startIndexStr); err == nil {
			startIndex = val
		}
	}

	if endIndexStr != "" {
		if val, err := strconv.Atoi(endIndexStr); err == nil {
			endIndex = val
		}
	}

	logs := lc.logger.GetEntries(startIndex, endIndex)

	response := map[string]interface{}{
		"logs": logs,
	}

	// Check if client accepts HTML (browser)
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		err := lc.renderer.Render(w, "logs", response)
		if err != nil {
			lc.logger.Errorf("Failed to render logs: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
