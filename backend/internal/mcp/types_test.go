package mcp

import (
	"encoding/json"
	"testing"
)

func TestRequestID_String(t *testing.T) {
	strID := NewStringRequestID("test-123")
	if strID.String() != "test-123" {
		t.Errorf("Expected 'test-123', got '%s'", strID.String())
	}

	intID := NewIntRequestID(456)
	if intID.String() != "456" {
		t.Errorf("Expected '456', got '%s'", intID.String())
	}
}

func TestRequestID_MarshalJSON(t *testing.T) {
	strID := NewStringRequestID("test-123")
	data, err := json.Marshal(strID)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(data) != `"test-123"` {
		t.Errorf("Expected '\"test-123\"', got '%s'", string(data))
	}

	intID := NewIntRequestID(456)
	data, err = json.Marshal(intID)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(data) != "456" {
		t.Errorf("Expected '456', got '%s'", string(data))
	}
}

func TestRequestID_UnmarshalJSON(t *testing.T) {
	var strID RequestID
	if err := json.Unmarshal([]byte(`"test-123"`), &strID); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if strID.String() != "test-123" {
		t.Errorf("Expected 'test-123', got '%s'", strID.String())
	}

	var intID RequestID
	if err := json.Unmarshal([]byte("456"), &intID); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if intID.String() != "456" {
		t.Errorf("Expected '456', got '%s'", intID.String())
	}
}

func TestJSONRPCRequest_MarshalJSON(t *testing.T) {
	params, _ := json.Marshal(map[string]string{"key": "value"})
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      NewStringRequestID("1"),
		Method:  "test.method",
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got '%v'", decoded["jsonrpc"])
	}
	if decoded["method"] != "test.method" {
		t.Errorf("Expected method 'test.method', got '%v'", decoded["method"])
	}
}

func TestJSONRPCResponse_MarshalJSON(t *testing.T) {
	result, _ := json.Marshal(map[string]string{"result": "success"})
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      NewStringRequestID("1"),
		Result:  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got '%v'", decoded["jsonrpc"])
	}
}

func TestJSONRPCResponse_WithError(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      NewStringRequestID("1"),
		Error: &JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	errData := decoded["error"].(map[string]interface{})
	if errData["code"].(float64) != -32600 {
		t.Errorf("Expected error code -32600, got %v", errData["code"])
	}
}

func TestJSONRPCError_Error(t *testing.T) {
	err := &JSONRPCError{
		Code:    -32600,
		Message: "Invalid Request",
	}

	if err.Error() != "Invalid Request" {
		t.Errorf("Expected 'Invalid Request', got '%s'", err.Error())
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrorCodeMethodNotFound, "Method not found", "test.method")

	if err.Code != ErrorCodeMethodNotFound {
		t.Errorf("Expected code %d, got %d", ErrorCodeMethodNotFound, err.Code)
	}
	if err.Message != "Method not found" {
		t.Errorf("Expected message 'Method not found', got '%s'", err.Message)
	}
	if err.Data != "test.method" {
		t.Errorf("Expected data 'test.method', got '%v'", err.Data)
	}
}

func TestTool_MarshalJSON(t *testing.T) {
	schema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]string{"type": "string"},
		},
	})

	tool := Tool{
		Name:        "search",
		Description: "Search for items",
		InputSchema: schema,
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["name"] != "search" {
		t.Errorf("Expected name 'search', got '%v'", decoded["name"])
	}
}

func TestCallToolParams_MarshalJSON(t *testing.T) {
	params := CallToolParams{
		Name: "search",
		Arguments: map[string]any{
			"query": "test query",
			"limit": 10,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["name"] != "search" {
		t.Errorf("Expected name 'search', got '%v'", decoded["name"])
	}

	args := decoded["arguments"].(map[string]interface{})
	if args["query"] != "test query" {
		t.Errorf("Expected query 'test query', got '%v'", args["query"])
	}
}

func TestResource_MarshalJSON(t *testing.T) {
	resource := Resource{
		URI:         "file:///path/to/file.txt",
		Name:        "file.txt",
		Description: "A text file",
		MimeType:    "text/plain",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["uri"] != "file:///path/to/file.txt" {
		t.Errorf("Expected uri 'file:///path/to/file.txt', got '%v'", decoded["uri"])
	}
}

func TestNewTextContent(t *testing.T) {
	content := NewTextContent("Hello, World!")

	if content.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", content.Type)
	}
	if content.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", content.Text)
	}
}

func TestNewImageContent(t *testing.T) {
	content := NewImageContent("image/png", "base64data")

	if content.Type != "image" {
		t.Errorf("Expected type 'image', got '%s'", content.Type)
	}
	if content.MimeType != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got '%s'", content.MimeType)
	}
	if content.Data != "base64data" {
		t.Errorf("Expected data 'base64data', got '%s'", content.Data)
	}
}

func TestNewResourceContent(t *testing.T) {
	resource := &ResourceContent{
		URI:      "file:///test.txt",
		MimeType: "text/plain",
		Text:     "content",
	}
	content := NewResourceContent(resource)

	if content.Type != "resource" {
		t.Errorf("Expected type 'resource', got '%s'", content.Type)
	}
	if content.Resource == nil {
		t.Error("Expected resource to be set")
	}
	if content.Resource.URI != "file:///test.txt" {
		t.Errorf("Expected URI 'file:///test.txt', got '%s'", content.Resource.URI)
	}
}

func TestInitializeParams_MarshalJSON(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: MCPVersion,
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["protocolVersion"] != MCPVersion {
		t.Errorf("Expected protocolVersion '%s', got '%v'", MCPVersion, decoded["protocolVersion"])
	}

	clientInfo := decoded["clientInfo"].(map[string]interface{})
	if clientInfo["name"] != "test-client" {
		t.Errorf("Expected client name 'test-client', got '%v'", clientInfo["name"])
	}
}

func TestPrompt_MarshalJSON(t *testing.T) {
	prompt := Prompt{
		Name:        "greeting",
		Description: "A greeting prompt",
		Arguments: []PromptArgument{
			{Name: "name", Description: "Name to greet", Required: true},
		},
	}

	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["name"] != "greeting" {
		t.Errorf("Expected name 'greeting', got '%v'", decoded["name"])
	}

	args := decoded["arguments"].([]interface{})
	if len(args) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(args))
	}
}

func TestLogLevel_Constants(t *testing.T) {
	levels := []LogLevel{
		LogLevelDebug,
		LogLevelInfo,
		LogLevelNotice,
		LogLevelWarning,
		LogLevelError,
		LogLevelCritical,
		LogLevelAlert,
		LogLevelEmergency,
	}

	expected := []string{
		"debug", "info", "notice", "warning",
		"error", "critical", "alert", "emergency",
	}

	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("Expected '%s', got '%s'", expected[i], string(level))
		}
	}
}
