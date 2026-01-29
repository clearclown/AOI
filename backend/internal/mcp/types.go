package mcp

import (
	"encoding/json"
)

// MCP Protocol Version
const MCPVersion = "2024-11-05"

// JSONRPCVersion is the JSON-RPC version used by MCP
const JSONRPCVersion = "2.0"

// ============================================================================
// Base JSON-RPC Types
// ============================================================================

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID)
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RequestID can be a string or integer
type RequestID struct {
	value interface{}
}

// NewStringRequestID creates a string request ID
func NewStringRequestID(id string) RequestID {
	return RequestID{value: id}
}

// NewIntRequestID creates an integer request ID
func NewIntRequestID(id int64) RequestID {
	return RequestID{value: id}
}

// MarshalJSON implements json.Marshaler
func (r RequestID) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RequestID) UnmarshalJSON(data []byte) error {
	var stringID string
	if err := json.Unmarshal(data, &stringID); err == nil {
		r.value = stringID
		return nil
	}
	var intID int64
	if err := json.Unmarshal(data, &intID); err == nil {
		r.value = intID
		return nil
	}
	return json.Unmarshal(data, &r.value)
}

// String returns the string representation of the request ID
func (r RequestID) String() string {
	if s, ok := r.value.(string); ok {
		return s
	}
	data, _ := json.Marshal(r.value)
	return string(data)
}

// ============================================================================
// MCP Initialize Types
// ============================================================================

// InitializeParams represents parameters for the initialize request
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResult represents the result of an initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Implementation describes a client or server implementation
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
	Roots        *RootsCapability       `json:"roots,omitempty"`
}

// ServerCapabilities describes what the server supports
type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
}

// SamplingCapability indicates sampling support
type SamplingCapability struct{}

// RootsCapability indicates roots support
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// PromptsCapability indicates prompts support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resources support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability indicates tools support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ============================================================================
// MCP Tool Types
// ============================================================================

// Tool represents an MCP tool
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListToolsResult represents the result of listing tools
type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// CallToolParams represents parameters for calling a tool
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// CallToolResult represents the result of calling a tool
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ============================================================================
// MCP Resource Types
// ============================================================================

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult represents the result of listing resources
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

// ReadResourceParams represents parameters for reading a resource
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult represents the result of reading a resource
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// ResourceTemplate represents a resource template
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourceTemplatesResult represents the result of listing resource templates
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        *string            `json:"nextCursor,omitempty"`
}

// ============================================================================
// MCP Prompt Types
// ============================================================================

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents an argument to a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult represents the result of listing prompts
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

// GetPromptParams represents parameters for getting a prompt
type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult represents the result of getting a prompt
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string       `json:"role"` // "user" or "assistant"
	Content ContentBlock `json:"content"`
}

// ============================================================================
// Content Types
// ============================================================================

// ContentBlock represents a block of content
type ContentBlock struct {
	Type     string          `json:"type"` // "text", "image", "resource"
	Text     string          `json:"text,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	Data     string          `json:"data,omitempty"` // base64 for images
	Resource *ResourceContent `json:"resource,omitempty"`
}

// NewTextContent creates a text content block
func NewTextContent(text string) ContentBlock {
	return ContentBlock{
		Type: "text",
		Text: text,
	}
}

// NewImageContent creates an image content block
func NewImageContent(mimeType, base64Data string) ContentBlock {
	return ContentBlock{
		Type:     "image",
		MimeType: mimeType,
		Data:     base64Data,
	}
}

// NewResourceContent creates a resource content block
func NewResourceContent(resource *ResourceContent) ContentBlock {
	return ContentBlock{
		Type:     "resource",
		Resource: resource,
	}
}

// ============================================================================
// Logging Types
// ============================================================================

// LogLevel represents a log level
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// SetLevelParams represents parameters for setting log level
type SetLevelParams struct {
	Level LogLevel `json:"level"`
}

// LoggingMessageParams represents a logging message notification
type LoggingMessageParams struct {
	Level  LogLevel `json:"level"`
	Logger string   `json:"logger,omitempty"`
	Data   any      `json:"data"`
}

// ============================================================================
// Notification Types
// ============================================================================

// ProgressParams represents progress notification parameters
type ProgressParams struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"`
	Total         float64 `json:"total,omitempty"`
}

// CancelledParams represents a cancelled notification
type CancelledParams struct {
	RequestID RequestID `json:"requestId"`
	Reason    string    `json:"reason,omitempty"`
}

// ============================================================================
// Error Codes
// ============================================================================

const (
	// Standard JSON-RPC error codes
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603

	// MCP-specific error codes
	ErrorCodeResourceNotFound = -32002
	ErrorCodeToolNotFound     = -32003
	ErrorCodePromptNotFound   = -32004
)

// NewError creates a new JSON-RPC error
func NewError(code int, message string, data interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (e *JSONRPCError) Error() string {
	return e.Message
}
