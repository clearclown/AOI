package tailscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aoi-protocol/aoi/internal/identity"
)

func TestAuth_AuthenticateRequest(t *testing.T) {
	client := NewMockClient()
	client.SetSelf(&NodeInfo{
		ID:     "self-1",
		IPs:    []string{"100.64.0.1"},
		Online: true,
		Tags:   []string{"tag:aoi-agent"},
	})
	client.AddPeer(&NodeInfo{
		ID:     "peer-1",
		IPs:    []string{"100.64.0.2"},
		Online: true,
		Tags:   []string{"tag:aoi-agent"},
	})

	tests := []struct {
		name           string
		config         AuthConfig
		remoteAddr     string
		expectError    error
		expectNodeInfo bool
	}{
		{
			name: "valid Tailscale IP",
			config: AuthConfig{
				RequireAuth: true,
			},
			remoteAddr:     "100.64.0.2:12345",
			expectError:    nil,
			expectNodeInfo: true,
		},
		{
			name: "non-Tailscale IP with strict mode",
			config: AuthConfig{
				RequireAuth: true,
			},
			remoteAddr:     "192.168.1.1:12345",
			expectError:    ErrNotTailscaleRequest,
			expectNodeInfo: false,
		},
		{
			name: "non-Tailscale IP with development fallback",
			config: AuthConfig{
				RequireAuth:  true,
				FallbackMode: "development",
			},
			remoteAddr:     "127.0.0.1:12345",
			expectError:    nil,
			expectNodeInfo: true,
		},
		{
			name: "non-Tailscale IP without require auth",
			config: AuthConfig{
				RequireAuth: false,
			},
			remoteAddr:     "192.168.1.1:12345",
			expectError:    nil,
			expectNodeInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuth(client, tt.config, nil)

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			nodeInfo, err := auth.AuthenticateRequest(req)

			if tt.expectError != nil {
				if err != tt.expectError {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectNodeInfo && nodeInfo == nil {
				t.Error("expected nodeInfo, got nil")
			}
			if !tt.expectNodeInfo && nodeInfo != nil {
				t.Errorf("expected nil nodeInfo, got %+v", nodeInfo)
			}
		})
	}
}

func TestAuth_AllowedTags(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:     "peer-1",
		IPs:    []string{"100.64.0.2"},
		Online: true,
		Tags:   []string{"tag:production"},
	})
	client.AddPeer(&NodeInfo{
		ID:     "peer-2",
		IPs:    []string{"100.64.0.3"},
		Online: true,
		Tags:   []string{"tag:aoi-agent"},
	})

	auth := NewAuth(client, AuthConfig{
		RequireAuth: true,
		AllowedTags: []string{"tag:aoi-agent"},
	}, nil)

	// Request from peer without allowed tag
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "100.64.0.2:12345"

	_, err := auth.AuthenticateRequest(req)
	if err != ErrTagNotAllowed {
		t.Errorf("expected ErrTagNotAllowed, got %v", err)
	}

	// Request from peer with allowed tag
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "100.64.0.3:12345"

	nodeInfo, err := auth.AuthenticateRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if nodeInfo == nil {
		t.Error("expected nodeInfo, got nil")
	}
}

func TestAuth_Middleware(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:     "peer-1",
		IPs:    []string{"100.64.0.2"},
		Online: true,
		Tags:   []string{"tag:aoi-agent"},
	})

	registry := identity.NewAgentRegistry()
	auth := NewAuth(client, AuthConfig{
		RequireAuth:        true,
		AutoRegisterAgents: true,
	}, registry)

	// Test handler that checks context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodeInfo := GetNodeInfoFromContext(r.Context())
		if nodeInfo == nil {
			t.Error("expected nodeInfo in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	protectedHandler := auth.Middleware(handler)

	// Test with valid Tailscale request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "100.64.0.2:12345"
	rr := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test with invalid request
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr = httptest.NewRecorder()

	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestAuth_MapNodeToAgent(t *testing.T) {
	client := NewMockClient()
	auth := NewAuth(client, AuthConfig{}, nil)

	// Map node to agent
	auth.MapNodeToAgent("node-1", "agent-1")

	// Retrieve mapping
	agentID, ok := auth.GetAgentIDForNode("node-1")
	if !ok {
		t.Error("expected mapping to exist")
	}
	if agentID != "agent-1" {
		t.Errorf("expected agentID 'agent-1', got '%s'", agentID)
	}

	// Test non-existent mapping
	_, ok = auth.GetAgentIDForNode("non-existent")
	if ok {
		t.Error("expected mapping to not exist")
	}

	// Remove mapping
	auth.RemoveNodeMapping("node-1")
	_, ok = auth.GetAgentIDForNode("node-1")
	if ok {
		t.Error("expected mapping to be removed")
	}
}

func TestAuth_RequireTag(t *testing.T) {
	client := NewMockClient()
	auth := NewAuth(client, AuthConfig{}, nil)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply tag requirement middleware
	protectedHandler := auth.RequireTag("tag:admin")(handler)

	// Test without node info in context (should fail)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	// Test with node info but without required tag
	nodeInfo := &NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:user"},
	}
	ctx := context.WithValue(context.Background(), ContextKeyNodeInfo, nodeInfo)
	req = httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rr = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	// Test with required tag
	nodeInfo.Tags = []string{"tag:admin", "tag:user"}
	ctx = context.WithValue(context.Background(), ContextKeyNodeInfo, nodeInfo)
	req = httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rr = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestGetNodeInfoFromContext(t *testing.T) {
	// Test with no node info
	ctx := context.Background()
	nodeInfo := GetNodeInfoFromContext(ctx)
	if nodeInfo != nil {
		t.Error("expected nil nodeInfo from empty context")
	}

	// Test with node info
	expected := &NodeInfo{ID: "node-1"}
	ctx = context.WithValue(ctx, ContextKeyNodeInfo, expected)
	nodeInfo = GetNodeInfoFromContext(ctx)
	if nodeInfo == nil {
		t.Error("expected nodeInfo from context")
	}
	if nodeInfo.ID != expected.ID {
		t.Errorf("expected ID '%s', got '%s'", expected.ID, nodeInfo.ID)
	}
}

func TestGetAgentIDFromContext(t *testing.T) {
	// Test with no agent ID
	ctx := context.Background()
	agentID := GetAgentIDFromContext(ctx)
	if agentID != "" {
		t.Errorf("expected empty agentID, got '%s'", agentID)
	}

	// Test with agent ID
	ctx = context.WithValue(ctx, ContextKeyAgentID, "agent-1")
	agentID = GetAgentIDFromContext(ctx)
	if agentID != "agent-1" {
		t.Errorf("expected 'agent-1', got '%s'", agentID)
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expected   string
	}{
		{
			name:       "from RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "from RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For single IP",
			remoteAddr: "10.0.0.1:12345",
			xff:        "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For multiple IPs",
			remoteAddr: "10.0.0.1:12345",
			xff:        "192.168.1.1, 10.0.0.2, 10.0.0.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xri:        "192.168.1.1",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			result := extractClientIP(req)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"localhost", true},
		{"192.168.1.1", false},
		{"100.64.0.1", false},
	}

	for _, tt := range tests {
		result := isLocalhost(tt.ip)
		if result != tt.expected {
			t.Errorf("isLocalhost(%s) = %v, expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestHasAllowedTag(t *testing.T) {
	tests := []struct {
		nodeTags    []string
		allowedTags []string
		expected    bool
	}{
		{
			nodeTags:    []string{"tag:aoi-agent"},
			allowedTags: []string{"tag:aoi-agent"},
			expected:    true,
		},
		{
			nodeTags:    []string{"tag:other"},
			allowedTags: []string{"tag:aoi-agent"},
			expected:    false,
		},
		{
			nodeTags:    []string{"tag:a", "tag:b", "tag:aoi-agent"},
			allowedTags: []string{"tag:aoi-agent"},
			expected:    true,
		},
		{
			nodeTags:    []string{},
			allowedTags: []string{"tag:aoi-agent"},
			expected:    false,
		},
		{
			nodeTags:    []string{"tag:a"},
			allowedTags: []string{},
			expected:    false,
		},
	}

	for i, tt := range tests {
		result := hasAllowedTag(tt.nodeTags, tt.allowedTags)
		if result != tt.expected {
			t.Errorf("test %d: hasAllowedTag(%v, %v) = %v, expected %v",
				i, tt.nodeTags, tt.allowedTags, result, tt.expected)
		}
	}
}
