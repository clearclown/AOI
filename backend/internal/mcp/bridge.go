package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	aoicontext "github.com/aoi-protocol/aoi/internal/context"
)

// MCPBridge bridges AOI queries and MCP tool/resource interactions
type MCPBridge struct {
	mu            sync.RWMutex
	clients       map[string]*MCPClient // serverName -> client
	contextStore  *aoicontext.ContextStore
	toolMappings  map[string]ToolMapping  // toolName -> mapping
	resourceCache map[string]*CachedResource
	cacheTimeout  time.Duration
}

// ToolMapping defines how an AOI query maps to an MCP tool call
type ToolMapping struct {
	ServerName    string            `json:"server_name"`
	ToolName      string            `json:"tool_name"`
	Description   string            `json:"description"`
	ArgumentMap   map[string]string `json:"argument_map"` // AOI field -> MCP argument
	ResultHandler string            `json:"result_handler"` // How to process results
}

// CachedResource holds a cached MCP resource
type CachedResource struct {
	URI       string
	Content   []ResourceContent
	CachedAt  time.Time
	ExpiresAt time.Time
}

// BridgeConfig holds configuration for the MCP bridge
type BridgeConfig struct {
	CacheTimeout time.Duration        `json:"cache_timeout"`
	Servers      []ServerConfig       `json:"servers"`
	ToolMappings []ToolMappingConfig  `json:"tool_mappings"`
}

// ServerConfig holds configuration for an MCP server connection
type ServerConfig struct {
	Name          string `json:"name"`
	Transport     string `json:"transport"`
	Command       string `json:"command,omitempty"`
	Args          []string `json:"args,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	AutoConnect   bool   `json:"auto_connect"`
}

// ToolMappingConfig holds configuration for a tool mapping
type ToolMappingConfig struct {
	QueryPattern  string            `json:"query_pattern"`
	ServerName    string            `json:"server_name"`
	ToolName      string            `json:"tool_name"`
	ArgumentMap   map[string]string `json:"argument_map"`
	ResultHandler string            `json:"result_handler"`
}

// NewMCPBridge creates a new MCP bridge
func NewMCPBridge(contextStore *aoicontext.ContextStore) *MCPBridge {
	return &MCPBridge{
		clients:       make(map[string]*MCPClient),
		contextStore:  contextStore,
		toolMappings:  make(map[string]ToolMapping),
		resourceCache: make(map[string]*CachedResource),
		cacheTimeout:  5 * time.Minute,
	}
}

// Configure applies a configuration to the bridge
func (b *MCPBridge) Configure(config *BridgeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if config.CacheTimeout > 0 {
		b.cacheTimeout = config.CacheTimeout
	}

	// Configure tool mappings
	for _, mapping := range config.ToolMappings {
		b.toolMappings[mapping.QueryPattern] = ToolMapping{
			ServerName:    mapping.ServerName,
			ToolName:      mapping.ToolName,
			ArgumentMap:   mapping.ArgumentMap,
			ResultHandler: mapping.ResultHandler,
		}
	}

	return nil
}

// AddClient adds an MCP client for a server
func (b *MCPBridge) AddClient(name string, client *MCPClient) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[name] = client
}

// RemoveClient removes an MCP client
func (b *MCPBridge) RemoveClient(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	client, ok := b.clients[name]
	if !ok {
		return fmt.Errorf("client not found: %s", name)
	}

	if err := client.Disconnect(); err != nil {
		log.Printf("[MCPBridge] Error disconnecting client %s: %v", name, err)
	}

	delete(b.clients, name)
	return nil
}

// GetClient retrieves an MCP client by name
func (b *MCPBridge) GetClient(name string) (*MCPClient, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	client, ok := b.clients[name]
	return client, ok
}

// ListClients returns a list of connected client names
func (b *MCPBridge) ListClients() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	names := make([]string, 0, len(b.clients))
	for name := range b.clients {
		names = append(names, name)
	}
	return names
}

// ============================================================================
// AOI to MCP Translation
// ============================================================================

// AOIQuery represents an AOI query that can be translated to MCP
type AOIQuery struct {
	Query        string            `json:"query"`
	ContextScope string            `json:"context_scope,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

// AOIResponse represents the response from translating an MCP result
type AOIResponse struct {
	Answer     string            `json:"answer"`
	Confidence float64           `json:"confidence"`
	Sources    []string          `json:"sources,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

// TranslateQueryToToolCall attempts to translate an AOI query to an MCP tool call
func (b *MCPBridge) TranslateQueryToToolCall(query AOIQuery) (*ToolCallRequest, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Find a matching tool mapping
	for pattern, mapping := range b.toolMappings {
		if b.matchesPattern(query.Query, pattern) {
			args := b.extractArguments(query, mapping)
			return &ToolCallRequest{
				ServerName: mapping.ServerName,
				ToolName:   mapping.ToolName,
				Arguments:  args,
				Mapping:    mapping,
			}, nil
		}
	}

	return nil, fmt.Errorf("no tool mapping found for query: %s", query.Query)
}

// ToolCallRequest represents a translated tool call request
type ToolCallRequest struct {
	ServerName string
	ToolName   string
	Arguments  map[string]any
	Mapping    ToolMapping
}

// ExecuteToolCall executes an MCP tool call and returns the result
func (b *MCPBridge) ExecuteToolCall(ctx context.Context, req *ToolCallRequest) (*AOIResponse, error) {
	client, ok := b.GetClient(req.ServerName)
	if !ok {
		return nil, fmt.Errorf("MCP client not found: %s", req.ServerName)
	}

	if !client.IsConnected() {
		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
		}
	}

	result, err := client.CallTool(ctx, req.ToolName, req.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return b.translateToolResult(result, req.Mapping)
}

// translateToolResult converts an MCP tool result to an AOI response
func (b *MCPBridge) translateToolResult(result *CallToolResult, mapping ToolMapping) (*AOIResponse, error) {
	if result.IsError {
		return &AOIResponse{
			Answer:     "Error executing tool",
			Confidence: 0.0,
			Metadata: map[string]any{
				"error": true,
			},
		}, nil
	}

	// Extract text content
	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}

	answer := strings.Join(texts, "\n")
	
	return &AOIResponse{
		Answer:     answer,
		Confidence: 0.85,
		Sources:    []string{fmt.Sprintf("mcp:%s/%s", mapping.ServerName, mapping.ToolName)},
		Metadata: map[string]any{
			"tool":   mapping.ToolName,
			"server": mapping.ServerName,
		},
	}, nil
}

// ============================================================================
// MCP Resources to AOI Context
// ============================================================================

// FetchResourceAsContext fetches an MCP resource and stores it as context
func (b *MCPBridge) FetchResourceAsContext(ctx context.Context, serverName, uri string) error {
	client, ok := b.GetClient(serverName)
	if !ok {
		return fmt.Errorf("MCP client not found: %s", serverName)
	}

	if !client.IsConnected() {
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to MCP server: %w", err)
		}
	}

	contents, err := client.ReadResource(ctx, uri)
	if err != nil {
		return fmt.Errorf("failed to read resource: %w", err)
	}

	// Cache the resource
	b.mu.Lock()
	b.resourceCache[uri] = &CachedResource{
		URI:       uri,
		Content:   contents,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(b.cacheTimeout),
	}
	b.mu.Unlock()

	// Store as context entries
	for _, content := range contents {
		entry := &aoicontext.ContextEntry{
			Type:    aoicontext.ContextTypeProject,
			Source:  fmt.Sprintf("mcp:%s", serverName),
			Content: content.Text,
			Summary: fmt.Sprintf("MCP Resource: %s", uri),
			Topics:  []string{"mcp", "resource"},
			Metadata: map[string]any{
				"uri":       uri,
				"mime_type": content.MimeType,
				"server":    serverName,
			},
		}

		if err := b.contextStore.Store(entry); err != nil {
			log.Printf("[MCPBridge] Failed to store context entry: %v", err)
		}
	}

	return nil
}

// GetCachedResource retrieves a cached resource
func (b *MCPBridge) GetCachedResource(uri string) (*CachedResource, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cached, ok := b.resourceCache[uri]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		delete(b.resourceCache, uri)
		return nil, false
	}

	return cached, true
}

// SyncAllResources fetches all resources from connected servers and stores as context
func (b *MCPBridge) SyncAllResources(ctx context.Context) error {
	b.mu.RLock()
	clients := make(map[string]*MCPClient)
	for name, client := range b.clients {
		clients[name] = client
	}
	b.mu.RUnlock()

	var errors []string

	for name, client := range clients {
		if !client.IsConnected() {
			continue
		}

		resources, err := client.ListResources(ctx)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		for _, resource := range resources {
			if err := b.FetchResourceAsContext(ctx, name, resource.URI); err != nil {
				log.Printf("[MCPBridge] Failed to fetch resource %s: %v", resource.URI, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors syncing resources: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ============================================================================
// Discovery and Status
// ============================================================================

// DiscoveryResult holds the result of discovering MCP server capabilities
type DiscoveryResult struct {
	ServerName   string             `json:"server_name"`
	ServerInfo   *Implementation    `json:"server_info"`
	Tools        []Tool             `json:"tools"`
	Resources    []Resource         `json:"resources"`
	Prompts      []Prompt           `json:"prompts"`
	Capabilities *ServerCapabilities `json:"capabilities"`
}

// DiscoverServer discovers the capabilities of an MCP server
func (b *MCPBridge) DiscoverServer(ctx context.Context, serverName string) (*DiscoveryResult, error) {
	client, ok := b.GetClient(serverName)
	if !ok {
		return nil, fmt.Errorf("MCP client not found: %s", serverName)
	}

	if !client.IsConnected() {
		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	result := &DiscoveryResult{
		ServerName:   serverName,
		ServerInfo:   client.GetServerInfo(),
		Capabilities: client.GetCapabilities(),
	}

	// List tools
	if caps := client.GetCapabilities(); caps != nil && caps.Tools != nil {
		tools, err := client.ListTools(ctx)
		if err != nil {
			log.Printf("[MCPBridge] Failed to list tools: %v", err)
		} else {
			result.Tools = tools
		}
	}

	// List resources
	if caps := client.GetCapabilities(); caps != nil && caps.Resources != nil {
		resources, err := client.ListResources(ctx)
		if err != nil {
			log.Printf("[MCPBridge] Failed to list resources: %v", err)
		} else {
			result.Resources = resources
		}
	}

	// List prompts
	if caps := client.GetCapabilities(); caps != nil && caps.Prompts != nil {
		prompts, err := client.ListPrompts(ctx)
		if err != nil {
			log.Printf("[MCPBridge] Failed to list prompts: %v", err)
		} else {
			result.Prompts = prompts
		}
	}

	return result, nil
}

// GetStatus returns the status of all connected MCP servers
func (b *MCPBridge) GetStatus() map[string]any {
	b.mu.RLock()
	defer b.mu.RUnlock()

	servers := make([]map[string]any, 0)
	for name, client := range b.clients {
		serverStatus := map[string]any{
			"name":      name,
			"connected": client.IsConnected(),
		}
		if info := client.GetServerInfo(); info != nil {
			serverStatus["server_info"] = info
		}
		servers = append(servers, serverStatus)
	}

	return map[string]any{
		"servers":        servers,
		"server_count":   len(b.clients),
		"cached_resources": len(b.resourceCache),
		"tool_mappings":  len(b.toolMappings),
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

// matchesPattern checks if a query matches a pattern
func (b *MCPBridge) matchesPattern(query, pattern string) bool {
	// Simple contains check for now
	// Could be extended to support regex or more complex matching
	return strings.Contains(strings.ToLower(query), strings.ToLower(pattern))
}

// extractArguments extracts arguments from an AOI query based on the mapping
func (b *MCPBridge) extractArguments(query AOIQuery, mapping ToolMapping) map[string]any {
	args := make(map[string]any)

	// Default: pass the query as input
	args["query"] = query.Query
	args["input"] = query.Query

	// Add context scope if specified
	if query.ContextScope != "" {
		args["context"] = query.ContextScope
	}

	// Apply argument mapping
	for aoiField, mcpArg := range mapping.ArgumentMap {
		switch aoiField {
		case "query":
			args[mcpArg] = query.Query
		case "context_scope":
			args[mcpArg] = query.ContextScope
		default:
			if val, ok := query.Metadata[aoiField]; ok {
				args[mcpArg] = val
			}
		}
	}

	return args
}

// RegisterToolMapping adds a tool mapping
func (b *MCPBridge) RegisterToolMapping(pattern string, mapping ToolMapping) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.toolMappings[pattern] = mapping
}

// ============================================================================
// JSON-RPC Handler
// ============================================================================

// HandleJSONRPC handles JSON-RPC calls for MCP-related methods
func (b *MCPBridge) HandleJSONRPC(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "aoi.mcp.status":
		return b.GetStatus(), nil

	case "aoi.mcp.discover":
		var p struct {
			ServerName string `json:"server_name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return b.DiscoverServer(ctx, p.ServerName)

	case "aoi.mcp.tools":
		var p struct {
			ServerName string `json:"server_name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		client, ok := b.GetClient(p.ServerName)
		if !ok {
			return nil, fmt.Errorf("client not found: %s", p.ServerName)
		}
		return client.ListTools(ctx)

	case "aoi.mcp.call":
		var p struct {
			ServerName string         `json:"server_name"`
			ToolName   string         `json:"tool_name"`
			Arguments  map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		client, ok := b.GetClient(p.ServerName)
		if !ok {
			return nil, fmt.Errorf("client not found: %s", p.ServerName)
		}
		return client.CallTool(ctx, p.ToolName, p.Arguments)

	case "aoi.mcp.resources":
		var p struct {
			ServerName string `json:"server_name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		client, ok := b.GetClient(p.ServerName)
		if !ok {
			return nil, fmt.Errorf("client not found: %s", p.ServerName)
		}
		return client.ListResources(ctx)

	case "aoi.mcp.read":
		var p struct {
			ServerName string `json:"server_name"`
			URI        string `json:"uri"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		client, ok := b.GetClient(p.ServerName)
		if !ok {
			return nil, fmt.Errorf("client not found: %s", p.ServerName)
		}
		return client.ReadResource(ctx, p.URI)

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}
