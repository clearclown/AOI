package tailscale

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/aoi-protocol/aoi/internal/identity"
	"github.com/aoi-protocol/aoi/pkg/aoi"
)

// Auth errors
var (
	ErrNotTailscaleRequest = errors.New("request is not from Tailscale network")
	ErrUnauthorizedNode    = errors.New("node is not authorized")
	ErrMissingNodeID       = errors.New("node ID not found in request")
	ErrTagNotAllowed       = errors.New("node tag is not in allowed list")
)

// contextKey is used for context values
type contextKey string

const (
	// ContextKeyNodeInfo is the context key for NodeInfo
	ContextKeyNodeInfo contextKey = "tailscale_node_info"
	// ContextKeyAgentID is the context key for the mapped agent ID
	ContextKeyAgentID contextKey = "tailscale_agent_id"
)

// AuthConfig holds configuration for Tailscale authentication
type AuthConfig struct {
	// RequireAuth requires all requests to come from Tailscale network
	RequireAuth bool
	// AllowedTags is a list of Tailscale ACL tags that are allowed
	AllowedTags []string
	// FallbackMode defines behavior when Tailscale is not available
	// "development" - allow localhost connections
	// "strict" - reject all non-Tailscale connections
	FallbackMode string
	// AutoRegisterAgents automatically registers agents from Tailscale nodes
	AutoRegisterAgents bool
}

// Auth provides Tailscale-based authentication
type Auth struct {
	client       Client
	config       AuthConfig
	registry     *identity.AgentRegistry
	nodeToAgent  map[string]string // maps Tailscale Node ID to Agent ID
	mu           sync.RWMutex
}

// NewAuth creates a new Tailscale authenticator
func NewAuth(client Client, config AuthConfig, registry *identity.AgentRegistry) *Auth {
	return &Auth{
		client:      client,
		config:      config,
		registry:    registry,
		nodeToAgent: make(map[string]string),
	}
}

// MapNodeToAgent maps a Tailscale node ID to an agent ID
func (a *Auth) MapNodeToAgent(nodeID, agentID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nodeToAgent[nodeID] = agentID
}

// GetAgentIDForNode returns the agent ID mapped to a Tailscale node
func (a *Auth) GetAgentIDForNode(nodeID string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	agentID, ok := a.nodeToAgent[nodeID]
	return agentID, ok
}

// RemoveNodeMapping removes the mapping for a node
func (a *Auth) RemoveNodeMapping(nodeID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.nodeToAgent, nodeID)
}

// AuthenticateRequest verifies that a request comes from an authorized Tailscale node
func (a *Auth) AuthenticateRequest(r *http.Request) (*NodeInfo, error) {
	ctx := r.Context()

	// Extract client IP
	clientIP := extractClientIP(r)

	// Check if it's a Tailscale IP
	if !a.client.IsTailscaleIP(clientIP) {
		// Check fallback mode
		if a.config.FallbackMode == "development" && isLocalhost(clientIP) {
			return &NodeInfo{
				ID:       "localhost",
				Name:     "localhost",
				Hostname: "localhost",
				IPs:      []string{clientIP},
				Online:   true,
				Tags:     []string{"tag:development"},
			}, nil
		}

		if a.config.RequireAuth {
			return nil, ErrNotTailscaleRequest
		}

		// Return a placeholder for non-Tailscale requests when auth is not required
		return nil, nil
	}

	// Get node info by IP
	nodeInfo, err := a.client.GetPeerByIP(ctx, clientIP)
	if err != nil {
		return nil, err
	}

	// Verify node is online
	if !nodeInfo.Online {
		return nil, ErrUnauthorizedNode
	}

	// Check allowed tags if configured
	if len(a.config.AllowedTags) > 0 {
		if !hasAllowedTag(nodeInfo.Tags, a.config.AllowedTags) {
			return nil, ErrTagNotAllowed
		}
	}

	return nodeInfo, nil
}

// Middleware returns an HTTP middleware that authenticates requests via Tailscale
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodeInfo, err := a.AuthenticateRequest(r)
		if err != nil {
			log.Printf("Tailscale auth failed: %v (client: %s)", err, extractClientIP(r))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add node info to context if available
		ctx := r.Context()
		if nodeInfo != nil {
			ctx = context.WithValue(ctx, ContextKeyNodeInfo, nodeInfo)

			// Check for mapped agent ID
			if agentID, ok := a.GetAgentIDForNode(nodeInfo.ID); ok {
				ctx = context.WithValue(ctx, ContextKeyAgentID, agentID)
			} else if a.config.AutoRegisterAgents {
				// Auto-register agent from Tailscale node
				agentID := a.autoRegisterAgent(ctx, nodeInfo)
				if agentID != "" {
					ctx = context.WithValue(ctx, ContextKeyAgentID, agentID)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MiddlewareFunc returns an HTTP middleware function
func (a *Auth) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return a.Middleware(next).ServeHTTP
}

// autoRegisterAgent automatically registers an agent for a Tailscale node
func (a *Auth) autoRegisterAgent(ctx context.Context, nodeInfo *NodeInfo) string {
	if a.registry == nil {
		return ""
	}

	// Generate agent ID from node info
	agentID := "ts-" + nodeInfo.ID

	// Check if agent already exists
	if _, err := a.registry.GetAgent(agentID); err == nil {
		// Agent already exists
		a.MapNodeToAgent(nodeInfo.ID, agentID)
		return agentID
	}

	// Create new agent identity
	agent := &aoi.AgentIdentity{
		ID:       agentID,
		Role:     aoi.RoleEngineer, // Default role
		Owner:    nodeInfo.UserID,
		Status:   "online",
		Endpoint: nodeInfo.IPs[0] + ":8080", // Default endpoint
		Metadata: map[string]interface{}{
			"tailscale_node_id": nodeInfo.ID,
			"tailscale_name":    nodeInfo.Name,
			"tailscale_hostname": nodeInfo.Hostname,
			"tailscale_tags":    nodeInfo.Tags,
		},
	}

	if err := a.registry.Register(agent); err != nil {
		log.Printf("Failed to auto-register agent for node %s: %v", nodeInfo.ID, err)
		return ""
	}

	a.MapNodeToAgent(nodeInfo.ID, agentID)
	log.Printf("Auto-registered agent %s for Tailscale node %s (%s)", agentID, nodeInfo.ID, nodeInfo.Name)

	return agentID
}

// GetNodeInfoFromContext retrieves NodeInfo from the request context
func GetNodeInfoFromContext(ctx context.Context) *NodeInfo {
	nodeInfo, _ := ctx.Value(ContextKeyNodeInfo).(*NodeInfo)
	return nodeInfo
}

// GetAgentIDFromContext retrieves the agent ID from the request context
func GetAgentIDFromContext(ctx context.Context) string {
	agentID, _ := ctx.Value(ContextKeyAgentID).(string)
	return agentID
}

// extractClientIP extracts the client IP from an HTTP request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// isLocalhost checks if an IP is a localhost address
func isLocalhost(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ip == "localhost"
	}

	return parsedIP.IsLoopback()
}

// hasAllowedTag checks if a node has any of the allowed tags
func hasAllowedTag(nodeTags, allowedTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range nodeTags {
		tagSet[tag] = true
	}

	for _, allowed := range allowedTags {
		if tagSet[allowed] {
			return true
		}
	}

	return false
}

// ProtectedHandler wraps an http.Handler with Tailscale authentication
func (a *Auth) ProtectedHandler(handler http.Handler) http.Handler {
	return a.Middleware(handler)
}

// RequireTag returns middleware that requires a specific Tailscale tag
func (a *Auth) RequireTag(tag string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nodeInfo := GetNodeInfoFromContext(r.Context())
			if nodeInfo == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			hasTag := false
			for _, t := range nodeInfo.Tags {
				if t == tag {
					hasTag = true
					break
				}
			}

			if !hasTag {
				http.Error(w, "Forbidden: missing required tag", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
