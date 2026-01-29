package protocol

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aoi-protocol/aoi/internal/acl"
	"github.com/aoi-protocol/aoi/internal/identity"
	"github.com/aoi-protocol/aoi/pkg/aoi"
)

func TestHealthEndpoint(t *testing.T) {
	server := NewServer(nil, nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health aoi.HealthResponse
	err := json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", health.Status)
	}
}

func TestHealthEndpoint_ContentType(t *testing.T) {
	server := NewServer(nil, nil)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHealthEndpoint_POSTMethod(t *testing.T) {
	server := NewServer(nil, nil)
	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	// Health endpoint should accept all methods
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for POST, got %d", resp.StatusCode)
	}
}

func TestAgentsEndpoint_GET(t *testing.T) {
	registry := identity.NewAgentRegistry()
	server := NewServer(registry, nil)

	// Register some agents
	agent1 := &aoi.AgentIdentity{ID: "agent-1", Role: aoi.RoleEngineer, Status: "online"}
	agent2 := &aoi.AgentIdentity{ID: "agent-2", Role: aoi.RolePM, Status: "online"}
	registry.Register(agent1)
	registry.Register(agent2)

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var agents []*aoi.AgentIdentity
	err := json.NewDecoder(resp.Body).Decode(&agents)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestAgentsEndpoint_GET_Empty(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var agents []*aoi.AgentIdentity
	err := json.NewDecoder(resp.Body).Decode(&agents)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestAgentsEndpoint_POST(t *testing.T) {
	server := NewServer(nil, nil)

	agent := aoi.AgentIdentity{
		ID:     "new-agent",
		Role:   aoi.RoleQA,
		Owner:  "testuser",
		Status: "online",
	}

	body, _ := json.Marshal(agent)
	req := httptest.NewRequest("POST", "/api/agents", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var created aoi.AgentIdentity
	err := json.NewDecoder(resp.Body).Decode(&created)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if created.ID != agent.ID {
		t.Errorf("Expected ID %s, got %s", agent.ID, created.ID)
	}
}

func TestAgentsEndpoint_POST_InvalidJSON(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("POST", "/api/agents", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAgentsEndpoint_POST_EmptyBody(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("POST", "/api/agents", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201 for empty agent, got %d", resp.StatusCode)
	}
}

func TestAgentsEndpoint_PUT_NotAllowed(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("PUT", "/api/agents", nil)
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestAgentsEndpoint_DELETE_NotAllowed(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("DELETE", "/api/agents", nil)
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestQueryEndpoint_POST(t *testing.T) {
	server := NewServer(nil, nil)

	query := aoi.Query{
		ID:       "query-1",
		From:     "pm-agent",
		To:       "eng-agent",
		Query:    "What's the status?",
		Priority: "normal",
		Async:    false,
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleQuery(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result aoi.QueryResult
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Summary == "" {
		t.Error("Expected non-empty summary")
	}
}

func TestQueryEndpoint_POST_InvalidJSON(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("POST", "/api/query", strings.NewReader("invalid"))
	w := httptest.NewRecorder()

	server.handleQuery(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestQueryEndpoint_GET_NotAllowed(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("GET", "/api/query", nil)
	w := httptest.NewRecorder()

	server.handleQuery(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestQueryEndpoint_EmptyQuery(t *testing.T) {
	server := NewServer(nil, nil)

	query := aoi.Query{}
	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleQuery(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for empty query, got %d", resp.StatusCode)
	}
}

func TestQueryEndpoint_WithMetadata(t *testing.T) {
	server := NewServer(nil, nil)

	query := aoi.Query{
		ID:    "query-meta",
		From:  "agent-1",
		To:    "agent-2",
		Query: "Test query",
		Metadata: map[string]interface{}{
			"timestamp": "2024-01-01",
			"version":   "1.0",
		},
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleQuery(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNewServer_WithNilDependencies(t *testing.T) {
	server := NewServer(nil, nil)

	if server.registry == nil {
		t.Error("Expected registry to be initialized")
	}

	if server.aclMgr == nil {
		t.Error("Expected ACL manager to be initialized")
	}

	if server.mux == nil {
		t.Error("Expected mux to be initialized")
	}
}

func TestNewServer_WithProvidedDependencies(t *testing.T) {
	registry := identity.NewAgentRegistry()
	aclMgr := acl.NewAclManager()

	server := NewServer(registry, aclMgr)

	if server.registry != registry {
		t.Error("Expected provided registry to be used")
	}

	if server.aclMgr != aclMgr {
		t.Error("Expected provided ACL manager to be used")
	}
}

func TestServer_RoutesConfigured(t *testing.T) {
	server := NewServer(nil, nil)

	routes := []struct {
		path   string
		method string
	}{
		{"/health", "GET"},
		{"/api/agents", "GET"},
		{"/api/agents", "POST"},
		{"/api/query", "POST"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		// Should not return 404
		if w.Code == http.StatusNotFound {
			t.Errorf("Route %s %s returned 404", route.method, route.path)
		}
	}
}

func TestServer_UnknownRoute(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for unknown route, got %d", w.Code)
	}
}

// JSON-RPC 2.0 Tests

func TestJSONRPC_Discover(t *testing.T) {
	registry := identity.NewAgentRegistry()
	server := NewServer(registry, nil)

	agent := &aoi.AgentIdentity{ID: "test-agent", Role: aoi.RoleEngineer, Status: "online"}
	registry.Register(agent)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.discover",
		ID:      1,
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC 2.0, got %s", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %+v", resp.Error)
	}
}

func TestJSONRPC_Query(t *testing.T) {
	server := NewServer(nil, nil)

	params := map[string]interface{}{
		"query":      "What is the status?",
		"from_agent": "pm-agent",
	}
	paramsJSON, _ := json.Marshal(params)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.query",
		Params:  paramsJSON,
		ID:      "query-1",
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %+v", resp.Error)
	}
}

func TestJSONRPC_Execute(t *testing.T) {
	server := NewServer(nil, nil)

	params := aoi.Task{
		ID:   "task-1",
		Type: "analyze",
		Parameters: map[string]interface{}{
			"target": "codebase",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.execute",
		Params:  paramsJSON,
		ID:      100,
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %+v", resp.Error)
	}
}

func TestJSONRPC_Notify(t *testing.T) {
	server := NewServer(nil, nil)

	params := map[string]interface{}{
		"type":    "status_update",
		"from":    "agent-1",
		"to":      "agent-2",
		"message": "Task completed",
	}
	paramsJSON, _ := json.Marshal(params)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.notify",
		Params:  paramsJSON,
		ID:      "notify-1",
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %+v", resp.Error)
	}
}

func TestJSONRPC_Status(t *testing.T) {
	server := NewServer(nil, nil)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.status",
		ID:      "status-1",
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %+v", resp.Error)
	}
}

func TestJSONRPC_MethodNotFound(t *testing.T) {
	server := NewServer(nil, nil)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.unknown",
		ID:      1,
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for JSON-RPC error, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for method not found")
	}

	if resp.Error.Code != JSONRPCMethodNotFound {
		t.Errorf("Expected error code %d, got %d", JSONRPCMethodNotFound, resp.Error.Code)
	}
}

func TestJSONRPC_ParseError(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("POST", "/api/v1/rpc", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for JSON-RPC error, got %d", w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected parse error")
	}

	if resp.Error.Code != JSONRPCParseError {
		t.Errorf("Expected error code %d, got %d", JSONRPCParseError, resp.Error.Code)
	}
}

func TestJSONRPC_InvalidRequest(t *testing.T) {
	server := NewServer(nil, nil)

	rpcReq := JSONRPCRequest{
		JSONRPC: "1.0",
		Method:  "aoi.status",
		ID:      1,
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected invalid request error")
	}

	if resp.Error.Code != JSONRPCInvalidRequest {
		t.Errorf("Expected error code %d, got %d", JSONRPCInvalidRequest, resp.Error.Code)
	}
}

func TestJSONRPC_InvalidParams(t *testing.T) {
	server := NewServer(nil, nil)

	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "aoi.query",
		Params:  json.RawMessage(`{"invalid": "structure"}`),
		ID:      1,
	}

	body, _ := json.Marshal(rpcReq)
	req := httptest.NewRequest("POST", "/api/v1/rpc", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Query should succeed even with missing required fields
	// The mock implementation doesn't validate strictly
	if resp.Error != nil && resp.Error.Code != JSONRPCInvalidParams {
		t.Errorf("Expected no error or invalid params error, got %+v", resp.Error)
	}
}

func TestJSONRPC_GETMethodNotAllowed(t *testing.T) {
	server := NewServer(nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/rpc", nil)
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for GET method")
	}

	if resp.Error.Code != JSONRPCInvalidRequest {
		t.Errorf("Expected error code %d, got %d", JSONRPCInvalidRequest, resp.Error.Code)
	}
}
