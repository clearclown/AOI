package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// TransportType represents the type of transport for MCP communication
type TransportType string

const (
	// TransportStdio uses stdin/stdout for communication
	TransportStdio TransportType = "stdio"
	// TransportHTTP uses HTTP/SSE for communication
	TransportHTTP TransportType = "http"
)

// ClientConfig holds configuration for an MCP client
type ClientConfig struct {
	// Transport type (stdio or http)
	Transport TransportType `json:"transport"`
	
	// For stdio transport
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	
	// For HTTP transport
	BaseURL string `json:"base_url,omitempty"`
	
	// Client identification
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"`
	
	// Timeouts
	ConnectTimeout time.Duration `json:"connect_timeout"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Transport:      TransportStdio,
		ClientName:     "aoi-mcp-client",
		ClientVersion:  "1.0.0",
		ConnectTimeout: 30 * time.Second,
		RequestTimeout: 60 * time.Second,
	}
}

// MCPClient provides an interface to interact with MCP servers
type MCPClient struct {
	config    *ClientConfig
	
	// Connection state
	connected    bool
	serverInfo   *Implementation
	capabilities *ServerCapabilities
	
	// For stdio transport
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
	
	// For HTTP transport
	httpClient *http.Client
	
	// Request ID counter
	requestID int64
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Pending requests waiting for responses
	pending map[string]chan *JSONRPCResponse
}

// NewMCPClient creates a new MCP client with the given configuration
func NewMCPClient(config *ClientConfig) *MCPClient {
	if config == nil {
		config = DefaultClientConfig()
	}
	
	return &MCPClient{
		config:     config,
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		pending:    make(map[string]chan *JSONRPCResponse),
	}
}

// Connect establishes a connection to the MCP server
func (c *MCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}
	
	switch c.config.Transport {
	case TransportStdio:
		if err := c.connectStdio(ctx); err != nil {
			c.mu.Unlock()
			return fmt.Errorf("failed to connect via stdio: %w", err)
		}
	case TransportHTTP:
		if err := c.connectHTTP(ctx); err != nil {
			c.mu.Unlock()
			return fmt.Errorf("failed to connect via HTTP: %w", err)
		}
	default:
		c.mu.Unlock()
		return fmt.Errorf("unsupported transport type: %s", c.config.Transport)
	}
	c.mu.Unlock()
	
	// Send initialize request (outside of lock to avoid deadlock)
	if err := c.initialize(ctx); err != nil {
		c.mu.Lock()
		c.disconnect()
		c.mu.Unlock()
		return fmt.Errorf("failed to initialize: %w", err)
	}
	
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

// connectStdio starts the MCP server process and establishes stdio communication
func (c *MCPClient) connectStdio(ctx context.Context) error {
	if c.config.Command == "" {
		return fmt.Errorf("command is required for stdio transport")
	}
	
	c.cmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	c.cmd.Env = append(os.Environ(), c.config.Env...)
	
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	c.reader = bufio.NewReader(c.stdout)
	
	// Start response reader goroutine
	go c.readResponses()
	
	return nil
}

// connectHTTP establishes an HTTP connection to the MCP server
func (c *MCPClient) connectHTTP(ctx context.Context) error {
	if c.config.BaseURL == "" {
		return fmt.Errorf("base_url is required for HTTP transport")
	}
	
	// Test connection with a simple health check
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.BaseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Allow connection even if health check fails - server might not implement it
		return nil
	}
	defer resp.Body.Close()
	
	return nil
}

// initialize sends the MCP initialize request
func (c *MCPClient) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: MCPVersion,
		ClientInfo: Implementation{
			Name:    c.config.ClientName,
			Version: c.config.ClientVersion,
		},
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
		},
	}
	
	var result InitializeResult
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return err
	}
	
	c.serverInfo = &result.ServerInfo
	c.capabilities = &result.Capabilities
	
	// Send initialized notification
	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}
	
	return nil
}

// Disconnect closes the connection to the MCP server
func (c *MCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	return c.disconnect()
}

func (c *MCPClient) disconnect() error {
	if !c.connected {
		return nil
	}
	
	c.connected = false
	
	if c.config.Transport == TransportStdio && c.cmd != nil {
		if c.stdin != nil {
			c.stdin.Close()
		}
		if c.stdout != nil {
			c.stdout.Close()
		}
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	
	return nil
}

// IsConnected returns whether the client is connected
func (c *MCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetServerInfo returns information about the connected server
func (c *MCPClient) GetServerInfo() *Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// GetCapabilities returns the server's capabilities
func (c *MCPClient) GetCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// ============================================================================
// Tool Methods
// ============================================================================

// ListTools lists all available tools from the MCP server
func (c *MCPClient) ListTools(ctx context.Context) ([]Tool, error) {
	var result ListToolsResult
	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool calls a tool on the MCP server
func (c *MCPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}
	
	var result CallToolResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ============================================================================
// Resource Methods
// ============================================================================

// ListResources lists all available resources from the MCP server
func (c *MCPClient) ListResources(ctx context.Context) ([]Resource, error) {
	var result ListResourcesResult
	if err := c.call(ctx, "resources/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// ReadResource reads a resource from the MCP server
func (c *MCPClient) ReadResource(ctx context.Context, uri string) ([]ResourceContent, error) {
	params := ReadResourceParams{URI: uri}
	
	var result ReadResourceResult
	if err := c.call(ctx, "resources/read", params, &result); err != nil {
		return nil, err
	}
	return result.Contents, nil
}

// ListResourceTemplates lists all resource templates from the MCP server
func (c *MCPClient) ListResourceTemplates(ctx context.Context) ([]ResourceTemplate, error) {
	var result ListResourceTemplatesResult
	if err := c.call(ctx, "resources/templates/list", nil, &result); err != nil {
		return nil, err
	}
	return result.ResourceTemplates, nil
}

// ============================================================================
// Prompt Methods
// ============================================================================

// ListPrompts lists all available prompts from the MCP server
func (c *MCPClient) ListPrompts(ctx context.Context) ([]Prompt, error) {
	var result ListPromptsResult
	if err := c.call(ctx, "prompts/list", nil, &result); err != nil {
		return nil, err
	}
	return result.Prompts, nil
}

// GetPrompt retrieves a specific prompt from the MCP server
func (c *MCPClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	params := GetPromptParams{
		Name:      name,
		Arguments: arguments,
	}
	
	var result GetPromptResult
	if err := c.call(ctx, "prompts/get", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ============================================================================
// Low-level Communication
// ============================================================================

// call makes a JSON-RPC call and waits for the response
func (c *MCPClient) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.RLock()
	if !c.connected && method != "initialize" {
		c.mu.RUnlock()
		return fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()
	
	id := atomic.AddInt64(&c.requestID, 1)
	idStr := fmt.Sprintf("%d", id)
	
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
	}
	
	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      NewStringRequestID(idStr),
		Method:  method,
		Params:  paramsJSON,
	}
	
	switch c.config.Transport {
	case TransportStdio:
		return c.callStdio(ctx, &req, idStr, result)
	case TransportHTTP:
		return c.callHTTP(ctx, &req, result)
	default:
		return fmt.Errorf("unsupported transport: %s", c.config.Transport)
	}
}

// callStdio makes a call via stdio transport
func (c *MCPClient) callStdio(ctx context.Context, req *JSONRPCRequest, id string, result interface{}) error {
	// Create response channel
	respChan := make(chan *JSONRPCResponse, 1)
	
	c.mu.Lock()
	c.pending[id] = respChan
	c.mu.Unlock()
	
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()
	
	// Send request
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	if _, err := c.stdin.Write(append(reqData, '\n')); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}
	
	// Wait for response
	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-respChan:
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil && resp.Result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}
		return nil
	}
}

// callHTTP makes a call via HTTP transport
func (c *MCPClient) callHTTP(ctx context.Context, req *JSONRPCRequest, result interface{}) error {
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/rpc", bytes.NewReader(reqData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if rpcResp.Error != nil {
		return rpcResp.Error
	}
	
	if result != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}
	
	return nil
}

// notify sends a notification (no response expected)
func (c *MCPClient) notify(ctx context.Context, method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
	}
	
	notification := JSONRPCNotification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}
	
	switch c.config.Transport {
	case TransportStdio:
		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}
		if _, err := c.stdin.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write notification: %w", err)
		}
		return nil
	case TransportHTTP:
		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("failed to marshal notification: %w", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/rpc", bytes.NewReader(data))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	default:
		return fmt.Errorf("unsupported transport: %s", c.config.Transport)
	}
}

// readResponses reads responses from stdio transport
func (c *MCPClient) readResponses() {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "[MCPClient] Error reading response: %v\n", err)
			}
			return
		}
		
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		
		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "[MCPClient] Error unmarshaling response: %v\n", err)
			continue
		}
		
		// Route response to waiting caller
		idStr := resp.ID.String()
		c.mu.RLock()
		ch, ok := c.pending[idStr]
		c.mu.RUnlock()
		
		if ok {
			select {
			case ch <- &resp:
			default:
			}
		}
	}
}
