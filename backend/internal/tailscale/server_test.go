package tailscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aoi-protocol/aoi/internal/acl"
	"github.com/aoi-protocol/aoi/internal/config"
	"github.com/aoi-protocol/aoi/internal/identity"
)

func TestServer_Integration(t *testing.T) {
	// Create mock client
	mockClient := NewMockClient()
	mockClient.SetSelf(&NodeInfo{
		ID:       "self-1",
		Name:     "test-server",
		Hostname: "test-server",
		IPs:      []string{"100.64.0.1"},
		Online:   true,
		Tags:     []string{"tag:aoi-server"},
	})
	mockClient.AddPeer(&NodeInfo{
		ID:       "peer-1",
		Name:     "test-agent",
		Hostname: "test-agent",
		IPs:      []string{"100.64.0.2"},
		Online:   true,
		Tags:     []string{"tag:aoi-agent"},
	})

	// Create registry and ACL manager
	registry := identity.NewAgentRegistry()
	aclManager := acl.NewAclManager()

	// Create auth config
	authConfig := AuthConfig{
		RequireAuth:        true,
		AllowedTags:        []string{"tag:aoi-agent", "tag:aoi-server"},
		FallbackMode:       "development",
		AutoRegisterAgents: true,
	}
	auth := NewAuth(mockClient, authConfig, registry)

	// Create ACL config with default mappings
	aclConfig := ACLConfig{
		TagMappings: DefaultTagMappings(),
	}
	tsACL := NewACL(mockClient, aclManager, aclConfig)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodeInfo := GetNodeInfoFromContext(r.Context())
		if nodeInfo != nil {
			w.Write([]byte("Hello, " + nodeInfo.Name))
		} else {
			w.Write([]byte("Hello, unknown"))
		}
	})

	// Wrap with auth middleware
	protectedHandler := auth.Middleware(handler)

	// Test 1: Request from valid Tailscale peer
	t.Run("valid Tailscale peer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "100.64.0.2:12345"
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		body := rr.Body.String()
		if body != "Hello, test-agent" {
			t.Errorf("expected 'Hello, test-agent', got '%s'", body)
		}
	})

	// Test 2: Request from unauthorized IP
	t.Run("unauthorized IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	// Test 3: Request from localhost in development mode
	t.Run("localhost in development mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	// Test 4: Permission middleware
	t.Run("permission middleware", func(t *testing.T) {
		permHandler := tsACL.PermissionMiddleware("agents/agent-1", "write")(handler)

		// With proper tags
		nodeInfo := &NodeInfo{
			ID:   "node-1",
			Tags: []string{"tag:aoi-agent"},
		}
		ctx := context.WithValue(context.Background(), ContextKeyNodeInfo, nodeInfo)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		rr := httptest.NewRecorder()

		permHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	// Test 5: Auto-registration
	t.Run("auto-registration", func(t *testing.T) {
		// The middleware should have auto-registered the agent
		agents := registry.Discover()
		found := false
		for _, agent := range agents {
			if agent.TailscaleNodeID == "peer-1" || agent.ID == "ts-peer-1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected agent to be auto-registered")
		}
	})
}

func TestConvertTagMappings(t *testing.T) {
	cfgMappings := []config.TagMappingConfig{
		{
			Tag:        "tag:admin",
			Resources:  []string{"*"},
			Permission: "admin",
		},
		{
			Tag:        "tag:user",
			Resources:  []string{"users/*"},
			Permission: "read",
		},
	}

	mappings := convertTagMappings(cfgMappings)

	if len(mappings) != 2 {
		t.Errorf("expected 2 mappings, got %d", len(mappings))
	}

	if mappings[0].Tag != "tag:admin" {
		t.Errorf("expected tag 'tag:admin', got '%s'", mappings[0].Tag)
	}

	if mappings[0].Permission != acl.PermissionAdmin {
		t.Errorf("expected PermissionAdmin, got %v", mappings[0].Permission)
	}
}

func TestNewIntegration(t *testing.T) {
	registry := identity.NewAgentRegistry()
	aclManager := acl.NewAclManager()

	// Test with disabled config
	cfg := config.TailscaleConfig{
		Enabled: false,
	}
	integration, err := NewIntegration(cfg, registry, aclManager)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if integration != nil {
		t.Error("expected nil integration when disabled")
	}

	// Test with enabled config in development mode
	// Note: This might fail if tailscaled is not running, but should fall back
	cfg = config.TailscaleConfig{
		Enabled:      true,
		FallbackMode: "development",
	}
	integration, err = NewIntegration(cfg, registry, aclManager)
	if err != nil {
		t.Errorf("unexpected error in development mode: %v", err)
	}
	// In development mode without tailscaled, we should still get an integration with mock client
}

func TestHealthHandler(t *testing.T) {
	// Create a mock integration
	mockClient := NewMockClient()
	mockClient.SetConnected(true)
	mockClient.SetSelf(&NodeInfo{
		ID:     "self-1",
		IPs:    []string{"100.64.0.1"},
		Online: true,
	})

	registry := identity.NewAgentRegistry()
	aclManager := acl.NewAclManager()

	server := &Server{
		client: mockClient,
		config: ServerConfig{
			TailscaleConfig: config.TailscaleConfig{
				Enabled:     true,
				RequireAuth: true,
			},
		},
	}

	// Test healthy response
	handler := server.HealthHandler()
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if body != `{"status":"healthy","tailscale":"connected"}` {
		t.Errorf("unexpected body: %s", body)
	}

	// Test disconnected response
	mockClient.SetConnected(false)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body = rr.Body.String()
	if body != `{"status":"degraded","tailscale":"disconnected"}` {
		t.Errorf("unexpected body: %s", body)
	}

	// Test with Tailscale disabled
	server.config.TailscaleConfig.Enabled = false
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body = rr.Body.String()
	if body != `{"status":"healthy","tailscale":"disabled"}` {
		t.Errorf("unexpected body: %s", body)
	}

	_ = registry
	_ = aclManager
}

func TestServer_WrapHandler(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.AddPeer(&NodeInfo{
		ID:     "peer-1",
		IPs:    []string{"100.64.0.2"},
		Online: true,
		Tags:   []string{"tag:aoi-agent"},
	})
	registry := identity.NewAgentRegistry()

	auth := NewAuth(mockClient, AuthConfig{RequireAuth: true}, registry)
	aclManager := acl.NewAclManager()
	tsACL := NewACL(mockClient, aclManager, ACLConfig{})

	server := &Server{
		client: mockClient,
		auth:   auth,
		tsACL:  tsACL,
		config: ServerConfig{
			TailscaleConfig: config.TailscaleConfig{
				Enabled:     true,
				RequireAuth: true,
			},
		},
	}

	handlerCalled := false
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// When enabled and requireAuth, handler should be wrapped with auth
	wrapped := server.WrapHandler(originalHandler)

	// Test that wrapped handler rejects unauthorized requests
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected wrapped handler to reject unauthorized request, got status %d", rr.Code)
	}

	// Test that wrapped handler allows authorized requests
	handlerCalled = false
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "100.64.0.2:12345"
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected wrapped handler to allow authorized request, got status %d", rr.Code)
	}
	if !handlerCalled {
		t.Error("expected original handler to be called")
	}

	// When disabled, handler should not be wrapped (should allow all requests)
	server.config.TailscaleConfig.Enabled = false
	wrapped = server.WrapHandler(originalHandler)

	handlerCalled = false
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("expected original handler to be called when Tailscale is disabled")
	}
}
