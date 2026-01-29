package tailscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aoi-protocol/aoi/internal/acl"
	"github.com/aoi-protocol/aoi/internal/config"
	"github.com/aoi-protocol/aoi/internal/identity"
)

// Server errors
var (
	ErrTailscaleNotAvailable = errors.New("Tailscale is not available")
	ErrNoTailscaleIP         = errors.New("no Tailscale IP available")
)

// ServerConfig holds configuration for the Tailscale-enabled server
type ServerConfig struct {
	// TailscaleConfig from the main config
	TailscaleConfig config.TailscaleConfig
	// Port to listen on
	Port int
	// Handler is the HTTP handler to serve
	Handler http.Handler
	// Registry is the agent registry
	Registry *identity.AgentRegistry
	// ACLManager is the ACL manager
	ACLManager *acl.AclManager
}

// Server wraps an HTTP server with Tailscale integration
type Server struct {
	client     Client
	auth       *Auth
	tsACL      *ACL
	httpServer *http.Server
	config     ServerConfig
	listener   net.Listener
}

// NewServer creates a new Tailscale-enabled server
func NewServer(cfg ServerConfig) (*Server, error) {
	// Create Tailscale client
	var client Client
	var err error

	if cfg.TailscaleConfig.SocketPath != "" {
		client, err = NewLocalClient(WithSocketPath(cfg.TailscaleConfig.SocketPath))
	} else {
		client, err = NewLocalClient()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Tailscale client: %w", err)
	}

	// Create auth config from Tailscale config
	authConfig := AuthConfig{
		RequireAuth:        cfg.TailscaleConfig.RequireAuth,
		AllowedTags:        cfg.TailscaleConfig.AllowedTags,
		FallbackMode:       cfg.TailscaleConfig.FallbackMode,
		AutoRegisterAgents: cfg.TailscaleConfig.AutoRegisterAgents,
	}

	// Create auth instance
	auth := NewAuth(client, authConfig, cfg.Registry)

	// Create ACL config from tag mappings
	aclConfig := ACLConfig{
		TagMappings: convertTagMappings(cfg.TailscaleConfig.TagMappings),
	}
	if len(aclConfig.TagMappings) == 0 {
		aclConfig.TagMappings = DefaultTagMappings()
	}

	// Create ACL instance
	tsACL := NewACL(client, cfg.ACLManager, aclConfig)

	// Wrap handler with auth middleware if enabled
	handler := cfg.Handler
	if cfg.TailscaleConfig.Enabled && cfg.TailscaleConfig.RequireAuth {
		handler = auth.Middleware(handler)
	}

	return &Server{
		client: client,
		auth:   auth,
		tsACL:  tsACL,
		config: cfg,
		httpServer: &http.Server{
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}, nil
}

// convertTagMappings converts config tag mappings to internal format
func convertTagMappings(cfgMappings []config.TagMappingConfig) []TagPermissionMapping {
	mappings := make([]TagPermissionMapping, 0, len(cfgMappings))
	for _, m := range cfgMappings {
		mappings = append(mappings, TagPermissionMapping{
			Tag:        m.Tag,
			Resources:  m.Resources,
			Permission: acl.PermissionLevel(config.ParsePermission(m.Permission)),
		})
	}
	return mappings
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	addr, err := s.getListenAddress(ctx)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	log.Printf("Server listening on %s", addr)

	// Start serving in a goroutine
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// getListenAddress determines the address to listen on
func (s *Server) getListenAddress(ctx context.Context) (string, error) {
	port := s.config.Port
	if port == 0 {
		port = 8080
	}

	if s.config.TailscaleConfig.BindToTailscale {
		// Get Tailscale IP
		tsIP, err := s.getTailscaleIP(ctx)
		if err != nil {
			if s.config.TailscaleConfig.FallbackMode == "development" {
				log.Printf("Tailscale not available, falling back to 127.0.0.1")
				return fmt.Sprintf("127.0.0.1:%d", port), nil
			}
			return "", err
		}
		return fmt.Sprintf("%s:%d", tsIP, port), nil
	}

	return fmt.Sprintf("0.0.0.0:%d", port), nil
}

// getTailscaleIP returns the local Tailscale IP address
func (s *Server) getTailscaleIP(ctx context.Context) (string, error) {
	self, err := s.client.GetSelf(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTailscaleNotAvailable, err)
	}

	if len(self.IPs) == 0 {
		return "", ErrNoTailscaleIP
	}

	// Prefer IPv4
	for _, ip := range self.IPs {
		if !strings.Contains(ip, ":") {
			return ip, nil
		}
	}

	return self.IPs[0], nil
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// GetClient returns the Tailscale client
func (s *Server) GetClient() Client {
	return s.client
}

// GetAuth returns the Tailscale auth instance
func (s *Server) GetAuth() *Auth {
	return s.auth
}

// GetACL returns the Tailscale ACL instance
func (s *Server) GetACL() *ACL {
	return s.tsACL
}

// Address returns the address the server is listening on
func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// IsConnected checks if the server is connected to Tailscale
func (s *Server) IsConnected(ctx context.Context) bool {
	return s.client.IsConnected(ctx)
}

// HealthHandler returns an HTTP handler for health checks
func (s *Server) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		status := "healthy"
		tailscaleStatus := "unknown"

		if s.config.TailscaleConfig.Enabled {
			if s.client.IsConnected(ctx) {
				tailscaleStatus = "connected"
			} else {
				tailscaleStatus = "disconnected"
				if s.config.TailscaleConfig.RequireAuth &&
					s.config.TailscaleConfig.FallbackMode != "development" {
					status = "degraded"
				}
			}
		} else {
			tailscaleStatus = "disabled"
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"%s","tailscale":"%s"}`, status, tailscaleStatus)
	}
}

// WrapHandler wraps an HTTP handler with Tailscale authentication
func (s *Server) WrapHandler(handler http.Handler) http.Handler {
	if s.config.TailscaleConfig.Enabled && s.config.TailscaleConfig.RequireAuth {
		return s.auth.Middleware(handler)
	}
	return handler
}

// WrapHandlerFunc wraps an HTTP handler function with Tailscale authentication
func (s *Server) WrapHandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	if s.config.TailscaleConfig.Enabled && s.config.TailscaleConfig.RequireAuth {
		return s.auth.MiddlewareFunc(handler)
	}
	return handler
}

// RequirePermission returns middleware that requires a specific permission
func (s *Server) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return s.tsACL.PermissionMiddleware(resource, action)
}

// RequireTag returns middleware that requires a specific Tailscale tag
func (s *Server) RequireTag(tag string) func(http.Handler) http.Handler {
	return s.auth.RequireTag(tag)
}

// SyncACLs syncs Tailscale ACLs for all mapped nodes
func (s *Server) SyncACLs(ctx context.Context) error {
	// Get all node-to-agent mappings from auth
	// This is a simplified implementation - in practice you might want to
	// iterate over all known mappings
	return nil
}

// Integration provides a simple integration point for existing servers
type Integration struct {
	Client Client
	Auth   *Auth
	ACL    *ACL
}

// NewIntegration creates a new Tailscale integration from config
func NewIntegration(cfg config.TailscaleConfig, registry *identity.AgentRegistry, aclManager *acl.AclManager) (*Integration, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Create client
	var client Client
	var err error
	if cfg.SocketPath != "" {
		client, err = NewLocalClient(WithSocketPath(cfg.SocketPath))
	} else {
		client, err = NewLocalClient()
	}
	if err != nil {
		// In development mode, create a mock client
		if cfg.FallbackMode == "development" {
			log.Printf("Tailscale not available, using mock client for development")
			mockClient := NewMockClient()
			mockClient.SetSelf(&NodeInfo{
				ID:       "localhost",
				Name:     "localhost",
				Hostname: "localhost",
				IPs:      []string{"127.0.0.1"},
				Online:   true,
				Tags:     []string{"tag:development"},
			})
			client = mockClient
		} else {
			return nil, fmt.Errorf("failed to create Tailscale client: %w", err)
		}
	}

	// Create auth
	authConfig := AuthConfig{
		RequireAuth:        cfg.RequireAuth,
		AllowedTags:        cfg.AllowedTags,
		FallbackMode:       cfg.FallbackMode,
		AutoRegisterAgents: cfg.AutoRegisterAgents,
	}
	auth := NewAuth(client, authConfig, registry)

	// Create ACL
	aclConfig := ACLConfig{
		TagMappings: convertTagMappings(cfg.TagMappings),
	}
	if len(aclConfig.TagMappings) == 0 {
		aclConfig.TagMappings = DefaultTagMappings()
	}
	tsACL := NewACL(client, aclManager, aclConfig)

	return &Integration{
		Client: client,
		Auth:   auth,
		ACL:    tsACL,
	}, nil
}
