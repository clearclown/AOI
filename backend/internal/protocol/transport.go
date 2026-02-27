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
	"github.com/aoi-protocol/aoi/internal/h2a"
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
	h2aMgr      *h2a.H2AManager
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
	return NewServerFull(registry, aclMgr, contextAPI, mcpBridge, nil)
}

// NewServerFull creates a fully configured HTTP server including H2A support.
func NewServerFull(registry *identity.AgentRegistry, aclMgr *acl.AclManager, contextAPI *aoicontext.ContextAPI, mcpBridge *mcp.MCPBridge, h2aMgr *h2a.H2AManager) *Server {
	if registry == nil {
		registry = identity.NewAgentRegistry()
	}
	if aclMgr == nil {
		aclMgr = acl.NewAclManager()
	}
	if h2aMgr == nil {
		h2aMgr = h2a.NewH2AManager()
	}

	wsHub := NewWSHub(nil)

	// Wire up the WebSocket hub to the H2A manager for output broadcasting.
	h2aMgr.SetWSHub(wsHub)

	s := &Server{
		registry:    registry,
		aclMgr:      aclMgr,
		mux:         http.NewServeMux(),
		wsHub:       wsHub,
		contextAPI:  contextAPI,
		mcpBridge:   mcpBridge,
		approvalMgr: approval.NewApprovalManager(),
		auditLogger: audit.NewAuditLogger(),
		h2aMgr:      h2aMgr,
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
	case strings.HasPrefix(req.Method, "aoi.h2a"):
		s.handleH2ARPC(w, &req)
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

// handleH2ARPC routes aoi.h2a.* JSON-RPC methods.
func (s *Server) handleH2ARPC(w http.ResponseWriter, req *JSONRPCRequest) {
	switch req.Method {
	case "aoi.h2a.register":
		s.handleH2ARegister(w, req)
	case "aoi.h2a.sessions":
		s.handleH2ASessions(w, req)
	case "aoi.h2a.send":
		s.handleH2ASend(w, req)
	case "aoi.h2a.stream":
		s.handleH2AStream(w, req)
	case "aoi.h2a.stop":
		s.handleH2AStop(w, req)
	default:
		s.sendJSONRPCError(w, req.ID, JSONRPCMethodNotFound, "Method not found", req.Method)
	}
}

// handleH2ARegister implements aoi.h2a.register — links an agent ID to a tmux session.
func (s *Server) handleH2ARegister(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		AgentID     string `json:"agent_id"`
		SessionName string `json:"session_name"`
		PaneName    string `json:"pane_name,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}
	if err := s.h2aMgr.RegisterSession(params.AgentID, params.SessionName, params.PaneName); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, err.Error(), nil)
		return
	}
	s.sendJSONRPCSuccess(w, req.ID, map[string]interface{}{
		"status":   "registered",
		"agent_id": params.AgentID,
	})
}

// handleH2ASessions implements aoi.h2a.sessions — returns all registered tmux sessions.
func (s *Server) handleH2ASessions(w http.ResponseWriter, req *JSONRPCRequest) {
	sessions := s.h2aMgr.ListSessions()
	s.sendJSONRPCSuccess(w, req.ID, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handleH2ASend implements aoi.h2a.send — sends a command and optionally returns captured output.
func (s *Server) handleH2ASend(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		TargetAgentID string `json:"target_agent_id"`
		FromUser      string `json:"from_user"`
		Command       string `json:"command"`
		CaptureOutput bool   `json:"capture_output"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	// ACL check: can from_user send to target?
	if !s.h2aMgr.CanSendTo(params.FromUser, params.TargetAgentID) {
		s.sendJSONRPCError(w, req.ID, JSONRPCACLDenied,
			fmt.Sprintf("user '%s' is not allowed to send to agent '%s'", params.FromUser, params.TargetAgentID),
			nil)
		return
	}

	result, err := s.h2aMgr.SendCommand(params.TargetAgentID, params.Command, params.CaptureOutput)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	log.Printf("[H2A] %s -> %s: %q", params.FromUser, params.TargetAgentID, params.Command)
	s.sendJSONRPCSuccess(w, req.ID, result)
}

// handleH2AStream implements aoi.h2a.stream — sends a command and begins output streaming via WebSocket.
func (s *Server) handleH2AStream(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		TargetAgentID string `json:"target_agent_id"`
		FromUser      string `json:"from_user"`
		Command       string `json:"command"`
		IntervalMs    int    `json:"interval_ms,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}

	// ACL check
	if !s.h2aMgr.CanSendTo(params.FromUser, params.TargetAgentID) {
		s.sendJSONRPCError(w, req.ID, JSONRPCACLDenied,
			fmt.Sprintf("user '%s' is not allowed to send to agent '%s'", params.FromUser, params.TargetAgentID),
			nil)
		return
	}

	// Send command first
	if _, err := s.h2aMgr.SendCommand(params.TargetAgentID, params.Command, false); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	// Start streaming output via WebSocket
	streamID, err := s.h2aMgr.StartStream(params.TargetAgentID, params.IntervalMs)
	if err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}

	log.Printf("[H2A] stream %s: %s -> %s: %q", streamID, params.FromUser, params.TargetAgentID, params.Command)
	s.sendJSONRPCSuccess(w, req.ID, map[string]interface{}{
		"status":    "streaming",
		"stream_id": streamID,
		"topic":     "h2a:" + params.TargetAgentID,
	})
}

// handleH2AStop implements aoi.h2a.stop — cancels an active stream.
func (s *Server) handleH2AStop(w http.ResponseWriter, req *JSONRPCRequest) {
	var params struct {
		StreamID string `json:"stream_id"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInvalidParams, "Invalid params", err.Error())
		return
	}
	if err := s.h2aMgr.StopStream(params.StreamID); err != nil {
		s.sendJSONRPCError(w, req.ID, JSONRPCInternalError, err.Error(), nil)
		return
	}
	s.sendJSONRPCSuccess(w, req.ID, map[string]string{"status": "stopped"})
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
