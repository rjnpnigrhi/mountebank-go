package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mountebank-testing/mountebank-go/internal/util"
)

// LogsController handles logs endpoints
type LogsController struct {
	logger *util.Logger
}

// NewLogsController creates a new logs controller
func NewLogsController(logger *util.Logger) *LogsController {
	return &LogsController{
		logger: logger,
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
