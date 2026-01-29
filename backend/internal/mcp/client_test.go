package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.Transport != TransportStdio {
		t.Errorf("Expected transport 'stdio', got '%s'", config.Transport)
	}
	if config.ClientName != "aoi-mcp-client" {
		t.Errorf("Expected client name 'aoi-mcp-client', got '%s'", config.ClientName)
	}
	if config.ClientVersion != "1.0.0" {
		t.Errorf("Expected client version '1.0.0', got '%s'", config.ClientVersion)
	}
	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected connect timeout 30s, got %v", config.ConnectTimeout)
	}
	if config.RequestTimeout != 60*time.Second {
		t.Errorf("Expected request timeout 60s, got %v", config.RequestTimeout)
	}
}

func TestNewMCPClient(t *testing.T) {
	config := &ClientConfig{
		Transport:     TransportHTTP,
		BaseURL:       "http://localhost:8080",
		ClientName:    "test-client",
		ClientVersion: "1.0.0",
	}

	client := NewMCPClient(config)
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.config != config {
		t.Error("Expected client to have the config reference")
	}
	if client.IsConnected() {
		t.Error("Expected client to not be connected initially")
	}
}

func TestNewMCPClient_NilConfig(t *testing.T) {
	client := NewMCPClient(nil)
	if client == nil {
		t.Fatal("Expected client to be created with default config")
	}

	if client.config.ClientName != "aoi-mcp-client" {
		t.Error("Expected default config to be used")
	}
}

func TestMCPClient_ConnectHTTP(t *testing.T) {
	// Create a mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/rpc":
			var req JSONRPCRequest
			json.NewDecoder(r.Body).Decode(&req)

			var result interface{}
			switch req.Method {
			case "initialize":
				result = InitializeResult{
					ProtocolVersion: MCPVersion,
					ServerInfo: Implementation{
						Name:    "test-server",
						Version: "1.0.0",
					},
					Capabilities: ServerCapabilities{
						Tools: &ToolsCapability{},
					},
				}
			case "notifications/initialized":
				// Notification, no response needed
				return
			}

			resultJSON, _ := json.Marshal(result)
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultJSON,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := &ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test-client",
		ClientVersion:  "1.0.0",
		RequestTimeout: 5 * time.Second,
	}

	client := NewMCPClient(config)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	serverInfo := client.GetServerInfo()
	if serverInfo == nil {
		t.Fatal("Expected server info to be set")
	}
	if serverInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", serverInfo.Name)
	}
}

func TestMCPClient_ListTools(t *testing.T) {
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
			schema, _ := json.Marshal(map[string]interface{}{"type": "object"})
			result = ListToolsResult{
				Tools: []Tool{
					{Name: "tool1", Description: "Tool 1", InputSchema: schema},
					{Name: "tool2", Description: "Tool 2", InputSchema: schema},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "tool1" {
		t.Errorf("Expected tool name 'tool1', got '%s'", tools[0].Name)
	}
}

func TestMCPClient_CallTool(t *testing.T) {
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
		case "tools/call":
			result = CallToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: "Tool executed successfully"},
				},
				IsError: false,
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	result, err := client.CallTool(ctx, "test-tool", map[string]any{"input": "test"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}
	if result.Content[0].Text != "Tool executed successfully" {
		t.Errorf("Expected text 'Tool executed successfully', got '%s'", result.Content[0].Text)
	}
}

func TestMCPClient_ListResources(t *testing.T) {
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
		case "resources/list":
			result = ListResourcesResult{
				Resources: []Resource{
					{URI: "file:///test.txt", Name: "test.txt", MimeType: "text/plain"},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	resources, err := client.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	if resources[0].URI != "file:///test.txt" {
		t.Errorf("Expected URI 'file:///test.txt', got '%s'", resources[0].URI)
	}
}

func TestMCPClient_ReadResource(t *testing.T) {
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
					{URI: "file:///test.txt", MimeType: "text/plain", Text: "File content"},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	contents, err := client.ReadResource(ctx, "file:///test.txt")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if len(contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(contents))
	}
	if contents[0].Text != "File content" {
		t.Errorf("Expected text 'File content', got '%s'", contents[0].Text)
	}
}

func TestMCPClient_ListPrompts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result interface{}
		switch req.Method {
		case "initialize":
			result = InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test", Version: "1.0"},
				Capabilities:    ServerCapabilities{Prompts: &PromptsCapability{}},
			}
		case "notifications/initialized":
			return
		case "prompts/list":
			result = ListPromptsResult{
				Prompts: []Prompt{
					{Name: "greeting", Description: "A greeting prompt"},
				},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(prompts))
	}
	if prompts[0].Name != "greeting" {
		t.Errorf("Expected prompt name 'greeting', got '%s'", prompts[0].Name)
	}
}

func TestMCPClient_Disconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		var result interface{}
		if req.Method == "initialize" {
			result = InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test", Version: "1.0"},
				Capabilities:    ServerCapabilities{},
			}
		}

		resultJSON, _ := json.Marshal(result)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	err := client.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("Expected client to be disconnected")
	}
}

func TestMCPClient_GetCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method == "initialize" {
			result := InitializeResult{
				ProtocolVersion: MCPVersion,
				ServerInfo:      Implementation{Name: "test", Version: "1.0"},
				Capabilities: ServerCapabilities{
					Tools:     &ToolsCapability{ListChanged: true},
					Resources: &ResourcesCapability{Subscribe: true},
					Prompts:   &PromptsCapability{},
				},
			}
			resultJSON, _ := json.Marshal(result)
			resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultJSON}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client := NewMCPClient(&ClientConfig{
		Transport:      TransportHTTP,
		BaseURL:        server.URL,
		ClientName:     "test",
		ClientVersion:  "1.0",
		RequestTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	client.Connect(ctx)

	caps := client.GetCapabilities()
	if caps == nil {
		t.Fatal("Expected capabilities to be set")
	}

	if caps.Tools == nil {
		t.Error("Expected Tools capability to be set")
	}
	if caps.Resources == nil {
		t.Error("Expected Resources capability to be set")
	}
	if caps.Prompts == nil {
		t.Error("Expected Prompts capability to be set")
	}
}
