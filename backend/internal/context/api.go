package context

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ContextAPI provides REST and JSON-RPC endpoints for context management
type ContextAPI struct {
	monitor *ContextMonitor
	store   *ContextStore
}

// NewContextAPI creates a new context API handler
func NewContextAPI(monitor *ContextMonitor, store *ContextStore) *ContextAPI {
	return &ContextAPI{
		monitor: monitor,
		store:   store,
	}
}

// RegisterRoutes registers the context API routes with an HTTP mux
func (api *ContextAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/context", api.handleContext)
	mux.HandleFunc("/api/v1/context/history", api.handleContextHistory)
	mux.HandleFunc("/api/v1/context/watch", api.handleContextWatch)
	mux.HandleFunc("/api/v1/context/stats", api.handleContextStats)
	mux.HandleFunc("/api/v1/context/activity", api.handleContextActivity)
}

// handleContext handles GET /api/v1/context - returns current context summary
func (api *ContextAPI) handleContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	summary := api.monitor.GetSummary()
	json.NewEncoder(w).Encode(summary)
}

// handleContextHistory handles GET /api/v1/context/history - returns context history
func (api *ContextAPI) handleContextHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	query := ContextQuery{}

	if project := r.URL.Query().Get("project"); project != "" {
		query.Project = project
	}
	if file := r.URL.Query().Get("file"); file != "" {
		query.File = file
	}
	if topic := r.URL.Query().Get("topic"); topic != "" {
		query.Topic = topic
	}
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		query.Type = ContextEntryType(typeStr)
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			query.Since = since
		}
	}
	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		if until, err := time.Parse(time.RFC3339, untilStr); err == nil {
			query.Until = until
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	history, err := api.store.Query(query)
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query context: %v", err))
		return
	}

	json.NewEncoder(w).Encode(history)
}

// handleContextWatch handles POST /api/v1/context/watch - adds a directory to watch
func (api *ContextAPI) handleContextWatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		var req WatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}

		resp, err := api.monitor.AddWatch(req)
		if err != nil {
			api.sendError(w, http.StatusBadRequest, fmt.Sprintf("Failed to add watch: %v", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)

	case http.MethodGet:
		// Return list of watched directories
		dirs := api.monitor.GetWatchedDirs()
		json.NewEncoder(w).Encode(map[string]any{
			"watched_dirs": dirs,
			"count":        len(dirs),
		})

	case http.MethodDelete:
		path := r.URL.Query().Get("path")
		if path == "" {
			api.sendError(w, http.StatusBadRequest, "Path parameter required")
			return
		}

		if err := api.monitor.RemoveWatch(path); err != nil {
			api.sendError(w, http.StatusBadRequest, fmt.Sprintf("Failed to remove watch: %v", err))
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "removed",
			"path":    path,
		})

	default:
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleContextStats handles GET /api/v1/context/stats - returns store statistics
func (api *ContextAPI) handleContextStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats := api.store.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// handleContextActivity handles POST /api/v1/context/activity - records manual activity
func (api *ContextAPI) handleContextActivity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Type        string         `json:"type"`
		Description string         `json:"description"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if req.Description == "" {
		api.sendError(w, http.StatusBadRequest, "Description is required")
		return
	}

	if err := api.monitor.RecordActivity(req.Type, req.Description, req.Metadata); err != nil {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to record activity: %v", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "recorded",
	})
}

// sendError sends a JSON error response
func (api *ContextAPI) sendError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error":   true,
		"message": message,
		"code":    code,
	})
}

// HandleJSONRPC handles JSON-RPC method calls for context
// This method is designed to be called from the main JSON-RPC handler
func (api *ContextAPI) HandleJSONRPC(method string, params json.RawMessage) (any, error) {
	switch method {
	case "aoi.context":
		return api.handleRPCContext(params)
	case "aoi.context.history":
		return api.handleRPCContextHistory(params)
	case "aoi.context.watch":
		return api.handleRPCContextWatch(params)
	case "aoi.context.activity":
		return api.handleRPCContextActivity(params)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// handleRPCContext handles the aoi.context JSON-RPC method
func (api *ContextAPI) handleRPCContext(params json.RawMessage) (any, error) {
	// No params needed for basic context summary
	return api.monitor.GetSummary(), nil
}

// handleRPCContextHistory handles the aoi.context.history JSON-RPC method
func (api *ContextAPI) handleRPCContextHistory(params json.RawMessage) (any, error) {
	var query ContextQuery
	if params != nil && len(params) > 0 {
		if err := json.Unmarshal(params, &query); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	return api.store.Query(query)
}

// handleRPCContextWatch handles the aoi.context.watch JSON-RPC method
func (api *ContextAPI) handleRPCContextWatch(params json.RawMessage) (any, error) {
	var req WatchRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return api.monitor.AddWatch(req)
}

// handleRPCContextActivity handles the aoi.context.activity JSON-RPC method
func (api *ContextAPI) handleRPCContextActivity(params json.RawMessage) (any, error) {
	var req struct {
		Type        string         `json:"type"`
		Description string         `json:"description"`
		Metadata    map[string]any `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if err := api.monitor.RecordActivity(req.Type, req.Description, req.Metadata); err != nil {
		return nil, err
	}

	return map[string]string{"status": "recorded"}, nil
}
