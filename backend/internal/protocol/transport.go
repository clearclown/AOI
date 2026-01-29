package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aoi-protocol/aoi/internal/acl"
	"github.com/aoi-protocol/aoi/internal/approval"
	"github.com/aoi-protocol/aoi/internal/audit"
	aoicontext "github.com/aoi-protocol/aoi/internal/context"
	"github.com/aoi-protocol/aoi/internal/identity"
	"github.com/aoi-protocol/aoi/internal/mcp"
	"github.com/aoi-protocol/aoi/internal/notify"
	"github.com/aoi-protocol/aoi/pkg/aoi"
)

// JSON-RPC 2.0 error codes
const (
	JSONRPCParseError     = -32700
	JSONRPCInvalidRequest = -32600
	JSONRPCMethodNotFound = -32601
	JSONRPCInvalidParams  = -32602
	JSONRPCInternalError  = -32603
	JSONRPCACLDenied      = -32000
	JSONRPCAgentNotFound  = -32001
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Server represents the HTTP server
type Server struct {
	registry    *identity.AgentRegistry
	aclMgr      *acl.AclManager
	mux         *http.ServeMux
	wsHub       *WSHub
	contextAPI  *aoicontext.ContextAPI
	mcpBridge   *mcp.MCPBridge
	approvalMgr *approval.ApprovalManager
	auditLogger *audit.AuditLogger
}

// NewServer creates a new HTTP server
func NewServer(registry *identity.AgentRegistry, aclMgr *acl.AclManager) *Server {
	return NewServerWithNotify(registry, aclMgr, nil)
}

// NewServerWithNotify creates a new HTTP server with a notification manager for WebSocket support
func NewServerWithNotify(registry *identity.AgentRegistry, aclMgr *acl.AclManager, notifyMgr *notify.NotificationManager) *Server {
	return NewServerWithContext(registry, aclMgr, nil, nil)
}

// NewServerWithContext creates a new HTTP server with context and MCP support
func NewServerWithContext(registry *identity.AgentRegistry, aclMgr *acl.AclManager, contextAPI *aoicontext.ContextAPI, mcpBridge *mcp.MCPBridge) *Server {
	if registry == nil {
		registry = identity.NewAgentRegistry()
	}
	if aclMgr == nil {
		aclMgr = acl.NewAclManager()
	}

	wsHub := NewWSHub(nil)

	s := &Server{
		registry:    registry,
		aclMgr:      aclMgr,
		mux:         http.NewServeMux(),
		wsHub:       wsHub,
		contextAPI:  contextAPI,
		mcpBridge:   mcpBridge,
		approvalMgr: approval.NewApprovalManager(),
		auditLogger: audit.NewAuditLogger(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Keep existing REST endpoints for backward compatibility
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/agents", s.handleAgents)
	s.mux.HandleFunc("/api/query", s.handleQuery)

	// Add JSON-RPC 2.0 endpoint
	s.mux.HandleFunc("/api/v1/rpc", s.handleJSONRPC)

	// Add WebSocket endpoint
	s.mux.HandleFunc("/api/v1/ws", s.HandleWebSocket(s.wsHub))

	// Register Context API routes if available
	if s.contextAPI != nil {
		s.contextAPI.RegisterRoutes(s.mux)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(aoi.HealthResponse{Status: "ok"})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		agents := s.registry.Discover()
		json.NewEncoder(w).Encode(agents)

	case http.MethodPost:
		var agent aoi.AgentIdentity
		if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.registry.Register(&agent); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(agent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var query aoi.Query
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock response for MVP
	result := aoi.QueryResult{
		Summary:   "Mock response: Work in progress",
		Progress:  50,
		Completed: true,
	}

	json.NewEncoder(w).Encode(result)
}

// handleJSONRPC processes JSON-RPC 2.0 requests
func (s *Server) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		s.sendJSONRPCError(w, nil, JSONRPCInvalidRequest, "Method not allowed", nil)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSONRPCError(w, nil, JSONRPCParseError, "Parse error", err.Error())
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidRequest, "Invalid JSON-RPC version", nil)
		return
	}

	// Route to appropriate method handler
	switch {
	case req.Method == "aoi.discover":
		s.handleDiscover(w, &req)
	case req.Method == "aoi.query":
		s.handleRPCQuery(w, &req)
	case req.Method == "aoi.execute":
		s.handleExecute(w, &req)
	case req.Method == "aoi.notify":
		s.handleNotify(w, &req)
	case req.Method == "aoi.status":
		s.handleStatus(w, &req)
	case strings.HasPrefix(req.Method, "aoi.context"):
		s.handleContextRPC(w, &req)
	case strings.HasPrefix(req.Method, "aoi.mcp"):
		s.handleMCPRPC(w, &req)
	case strings.HasPrefix(req.Method, "aoi.approval"):
		s.handleApprovalRPC(w, &req)
	case strings.HasPrefix(req.Method, "aoi.audit"):
		s.handleAuditRPC(w, &req)
	default:
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "Method not found", req.Method)
	}
}

// handleDiscover implements aoi.discover method
func (s *Server) handleDiscover(w http.ResponseWriter, req *JSONRPCRequest) {
	agents := s.registry.Discover()

	result := map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleRPCQuery implements aoi.query method
func (s *Server) handleRPCQuery(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		Query        string            `json:"query"`
		FromAgent    string            `json:"from_agent"`
		ContextScope string            `json:"context_scope,omitempty"`
		Metadata     map[string]string `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	// Mock response for now
	result := map[string]interface{}{
		"answer":     fmt.Sprintf("Response to: %s", params.Query),
		"confidence": 0.85,
		"sources":    []string{},
		"metadata":   params.Metadata,
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleExecute implements aoi.execute method
func (s *Server) handleExecute(w http.ResponseWriter, req *JSONRPCRequest) {
	var params aoi.Task

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	// Mock execution result
	result := aoi.TaskResult{
		TaskID: params.ID,
		Status: "completed",
		Output: "Task executed successfully",
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleNotify implements aoi.notify method (no response expected)
func (s *Server) handleNotify(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		Type    string                 `json:"type"`
		From    string                 `json:"from"`
		To      string                 `json:"to"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	// Notification accepted
	result := map[string]interface{}{
		"status": "accepted",
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleStatus implements aoi.status method
func (s *Server) handleStatus(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		AgentID string `json:"agent_id,omitempty"`
	}

	if req.Params != nil && len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
			return
		}
	}

	result := map[string]interface{}{
		"status": "online",
		"agents": len(s.registry.Discover()),
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleContextRPC routes context-related JSON-RPC methods
func (s *Server) handleContextRPC(w http.ResponseWriter, req *JSONRPCRequest) {
	if s.contextAPI == nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "Context API not available", nil)
		return
	}

	result, err := s.contextAPI.HandleJSONRPC(req.Method, req.Params)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleMCPRPC routes MCP-related JSON-RPC methods
func (s *Server) handleMCPRPC(w http.ResponseWriter, req *JSONRPCRequest) {
	if s.mcpBridge == nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "MCP Bridge not available", nil)
		return
	}

	ctx := context.Background()
	result, err := s.mcpBridge.HandleJSONRPC(ctx, req.Method, req.Params)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleApprovalRPC routes approval-related JSON-RPC methods
func (s *Server) handleApprovalRPC(w http.ResponseWriter, req *JSONRPCRequest) {
	if s.approvalMgr == nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "Approval Manager not available", nil)
		return
	}

	result, err := s.approvalMgr.HandleJSONRPC(req.Method, req.Params)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleAuditRPC routes audit-related JSON-RPC methods
func (s *Server) handleAuditRPC(w http.ResponseWriter, req *JSONRPCRequest) {
	if s.auditLogger == nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "Audit Logger not available", nil)
		return
	}

	result, err := s.auditLogger.HandleJSONRPC(req.Method, req.Params)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	s.sendJSONRPCSuccess(w, req.ID, result)
}

// sendJSONRPCSuccess sends a successful JSON-RPC response
func (s *Server) sendJSONRPCSuccess(w http.ResponseWriter, id interface{}, result interface{}) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		s.sendJSONRPCError(w, id, JSONRPCInternalError, "Failed to marshal result", err.Error())
		return
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  resultJSON,
		ID:      id,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// sendJSONRPCError sends a JSON-RPC error response
func (s *Server) sendJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still HTTP 200
	json.NewEncoder(w).Encode(resp)
}

// GetWSHub returns the WebSocket hub for external use
func (s *Server) GetWSHub() *WSHub {
	return s.wsHub
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	log.Printf("Starting server on %s", addr)

	// Start WebSocket hub in background
	go s.wsHub.Run()

	return http.ListenAndServe(addr, s.mux)
}
