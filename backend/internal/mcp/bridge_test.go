package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	aoicontext "github.com/aoi-protocol/aoi/internal/context"
)

func setupTestBridge() (*MCPBridge, *aoicontext.ContextStore, func()) {
	store := aoicontext.NewContextStore(1 * time.Hour)
	bridge := NewMCPBridge(store)

	cleanup := func() {
		store.Stop()
	}

	return bridge, store, cleanup
}

func TestNewMCPBridge(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	if bridge == nil {
		t.Fatal("Expected bridge to be created")
	}

	clients := bridge.ListClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(clients))
	}
}

func TestMCPBridge_AddRemoveClient(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	config := &ClientConfig{
		Transport:     TransportHTTP,
		BaseURL:       "http://localhost:8080",
		ClientName:    "test-client",
		ClientVersion: "1.0.0",
	}
	client := NewMCPClient(config)

	bridge.AddClient("test-server", client)

	clients := bridge.ListClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}

	retrieved, ok := bridge.GetClient("test-server")
	if !ok {
		t.Error("Expected to retrieve client")
	}
	if retrieved != client {
		t.Error("Expected retrieved client to match")
	}

	err := bridge.RemoveClient("test-server")
	if err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	clients = bridge.ListClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients after removal, got %d", len(clients))
	}
}

func TestMCPBridge_RemoveClient_NotFound(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	err := bridge.RemoveClient("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent client")
	}
}

func TestMCPBridge_Configure(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	config := &BridgeConfig{
		CacheTimeout: 10 * time.Minute,
		ToolMappings: []ToolMappingConfig{
			{
				QueryPattern: "search",
				ServerName:   "search-server",
				ToolName:     "search_tool",
				ArgumentMap:  map[string]string{"query": "input"},
			},
		},
	}

	err := bridge.Configure(config)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
}

func TestMCPBridge_RegisterToolMapping(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	mapping := ToolMapping{
		ServerName:  "test-server",
		ToolName:    "test-tool",
		Description: "A test tool",
		ArgumentMap: map[string]string{"query": "input"},
	}

	bridge.RegisterToolMapping("test-pattern", mapping)

	// Should be able to translate a query that matches the pattern
	query := AOIQuery{
		Query: "test-pattern search for something",
	}

	req, err := bridge.TranslateQueryToToolCall(query)
	if err != nil {
		t.Fatalf("TranslateQueryToToolCall failed: %v", err)
	}

	if req.ServerName != "test-server" {
		t.Errorf("Expected server 'test-server', got '%s'", req.ServerName)
	}
	if req.ToolName != "test-tool" {
		t.Errorf("Expected tool 'test-tool', got '%s'", req.ToolName)
	}
}

func TestMCPBridge_TranslateQueryToToolCall_NoMapping(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	query := AOIQuery{
		Query: "unknown query that has no mapping",
	}

	_, err := bridge.TranslateQueryToToolCall(query)
	if err == nil {
		t.Error("Expected error for query with no mapping")
	}
}

func TestMCPBridge_GetStatus(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	status := bridge.GetStatus()

	serverCount, ok := status["server_count"].(int)
	if !ok {
		t.Error("Expected server_count in status")
	}
	if serverCount != 0 {
		t.Errorf("Expected 0 servers, got %d", serverCount)
	}

	cachedResources, ok := status["cached_resources"].(int)
	if !ok {
		t.Error("Expected cached_resources in status")
	}
	if cachedResources != 0 {
		t.Errorf("Expected 0 cached resources, got %d", cachedResources)
	}
}

func TestMCPBridge_GetCachedResource_NotFound(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	_, ok := bridge.GetCachedResource("file:///nonexistent")
	if ok {
		t.Error("Expected resource to not be found")
	}
}

func TestMCPBridge_ExecuteToolCall_ClientNotFound(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	req := &ToolCallRequest{
		ServerName: "nonexistent-server",
		ToolName:   "test-tool",
		Arguments:  map[string]any{},
	}

	ctx := context.Background()
	_, err := bridge.ExecuteToolCall(ctx, req)
	if err == nil {
		t.Error("Expected error for nonexistent client")
	}
}

func TestMCPBridge_DiscoverServer(t *testing.T) {
	// Create a mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result interface{}
		switch req.Method {
		case "initialize":
			result = InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test-server", Version: "1.0.0"},
				Capabilities: ServerCapabilities{
					Tools:     &ToolsCapability{},
					Resources: &ResourcesCapability{},
				},
			}
		case "notifications/initialized":
			return
		case "tools/list":
			schema, _ := json.Marshal(map[string]interface{}{"type": "object"})
			result = ListToolsResult{
				Tools: []Tool{
					{Name: "tool1", Description: "Test tool", InputSchema: schema},
				},
			}
		case "resources/list":
			result = ListResourcesResult{
				Resources: []Resource{
					{URI: "file:///test.txt", Name: "test.txt"},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	config := &ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	}
	client := NewMCPClient(config)
	bridge.AddClient("test-server", client)

	ctx := context.Background()
	discovery, err := bridge.DiscoverServer(ctx, "test-server")
	if err != nil {
		t.Fatalf("DiscoverServer failed: %v", err)
	}

	if discovery.ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", discovery.ServerName)
	}
	if discovery.ServerInfo == nil {
		t.Error("Expected server info to be set")
	}
	if len(discovery.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(discovery.Tools))
	}
	if len(discovery.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(discovery.Resources))
	}
}

func TestMCPBridge_DiscoverServer_ClientNotFound(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	ctx := context.Background()
	_, err := bridge.DiscoverServer(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent client")
	}
}

func TestMCPBridge_FetchResourceAsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result interface{}
		switch req.Method {
		case "initialize":
			result = InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test", Version: "1.0"},
				Capabilities:    ServerCapabilities{Resources: &ResourcesCapability{}},
			}
		case "notifications/initialized":
			return
		case "resources/read":
			result = ReadResourceResult{
				Contents: []ResourceContent{
					{URI: "file:///test.txt", MimeType: "text/plain", Text: "Test content"},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bridge, store, cleanup := setupTestBridge()
	defer cleanup()

	config := &ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	}
	client := NewMCPClient(config)
	bridge.AddClient("test-server", client)

	ctx := context.Background()
	err := bridge.FetchResourceAsContext(ctx, "test-server", "file:///test.txt")
	if err != nil {
		t.Fatalf("FetchResourceAsContext failed: %v", err)
	}

	// Check that resource is cached
	cached, ok := bridge.GetCachedResource("file:///test.txt")
	if !ok {
		t.Error("Expected resource to be cached")
	}
	if cached.URI != "file:///test.txt" {
		t.Errorf("Expected URI 'file:///test.txt', got '%s'", cached.URI)
	}

	// Check that context entry was stored
	history, _ := store.Query(aoicontext.ContextQuery{Limit: 10})
	if len(history.Entries) == 0 {
		t.Error("Expected context entry to be stored")
	}
}

func TestMCPBridge_HandleJSONRPC_Status(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	ctx := context.Background()
	result, err := bridge.HandleJSONRPC(ctx, "aoi.mcp.status", nil)
	if err != nil {
		t.Fatalf("HandleJSONRPC failed: %v", err)
	}

	status, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map")
	}

	if _, ok := status["server_count"]; !ok {
		t.Error("Expected server_count in result")
	}
}

func TestMCPBridge_HandleJSONRPC_Discover(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result interface{}
		switch req.Method {
		case "initialize":
			result = InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test", Version: "1.0"},
				Capabilities:    ServerCapabilities{Tools: &ToolsCapability{}},
			}
		case "notifications/initialized":
			return
		case "tools/list":
			result = ListToolsResult{Tools: []Tool{}}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	config := &ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	}
	client := NewMCPClient(config)
	bridge.AddClient("test-server", client)

	ctx := context.Background()
	params, _ := json.Marshal(map[string]string{"server_name": "test-server"})
	result, err := bridge.HandleJSONRPC(ctx, "aoi.mcp.discover", params)
	if err != nil {
		t.Fatalf("HandleJSONRPC failed: %v", err)
	}

	discovery, ok := result.(*DiscoveryResult)
	if !ok {
		t.Fatal("Expected result to be *DiscoveryResult")
	}

	if discovery.ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", discovery.ServerName)
	}
}

func TestMCPBridge_HandleJSONRPC_UnknownMethod(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	ctx := context.Background()
	_, err := bridge.HandleJSONRPC(ctx, "aoi.mcp.unknown", nil)
	if err == nil {
		t.Error("Expected error for unknown method")
	}
}

func TestMCPBridge_TranslateToolResult(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	result := &CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Line 1"},
			{Type: "text", Text: "Line 2"},
		},
		IsError: false,
	}

	mapping := ToolMapping{
		ServerName: "test-server",
		ToolName:   "test-tool",
	}

	response, err := bridge.translateToolResult(result, mapping)
	if err != nil {
		t.Fatalf("translateToolResult failed: %v", err)
	}

	if response.Answer != "Line 1\nLine 2" {
		t.Errorf("Expected answer 'Line 1\\nLine 2', got '%s'", response.Answer)
	}
	if response.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", response.Confidence)
	}
}

func TestMCPBridge_TranslateToolResult_Error(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	result := &CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Error occurred"},
		},
		IsError: true,
	}

	mapping := ToolMapping{
		ServerName: "test-server",
		ToolName:   "test-tool",
	}

	response, err := bridge.translateToolResult(result, mapping)
	if err != nil {
		t.Fatalf("translateToolResult failed: %v", err)
	}

	if response.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for error, got %f", response.Confidence)
	}
}

func TestMCPBridge_ExtractArguments(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	query := AOIQuery{
		Query:        "test query",
		ContextScope: "project-scope",
		Metadata: map[string]any{
			"custom_field": "custom_value",
		},
	}

	mapping := ToolMapping{
		ArgumentMap: map[string]string{
			"query":         "search_query",
			"context_scope": "scope",
			"custom_field":  "custom",
		},
	}

	args := bridge.extractArguments(query, mapping)

	if args["search_query"] != "test query" {
		t.Errorf("Expected search_query 'test query', got '%v'", args["search_query"])
	}
	if args["scope"] != "project-scope" {
		t.Errorf("Expected scope 'project-scope', got '%v'", args["scope"])
	}
	if args["custom"] != "custom_value" {
		t.Errorf("Expected custom 'custom_value', got '%v'", args["custom"])
	}
}

func TestMCPBridge_MatchesPattern(t *testing.T) {
	bridge, _, cleanup := setupTestBridge()
	defer cleanup()

	tests := []struct {
		query    string
		pattern  string
		expected bool
	}{
		{"search for documents", "search", true},
		{"SEARCH for documents", "search", true},
		{"find files", "search", false},
		{"analyze code", "analyze", true},
		{"code analysis", "analyze", false},
	}

	for _, tt := range tests {
		result := bridge.matchesPattern(tt.query, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchesPattern(%q, %q) = %v, expected %v", tt.query, tt.pattern, result, tt.expected)
		}
	}
}
