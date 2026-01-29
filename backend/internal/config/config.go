package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the complete configuration for an AOI agent
type Config struct {
	Agent     AgentConfig     `json:"agent"`
	Network   NetworkConfig   `json:"network"`
	ACL       ACLConfig       `json:"acl"`
	Context   ContextConfig   `json:"context"`
	MCP       MCPConfig       `json:"mcp"`
	Tailscale TailscaleConfig `json:"tailscale"`
}

// AgentConfig contains agent identity configuration
type AgentConfig struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Owner string `json:"owner"`
}

// NetworkConfig contains network and transport configuration
type NetworkConfig struct {
	ListenAddr string `json:"listen_addr"`
	TLSEnabled bool   `json:"tls_enabled"`
}

// ACLConfig contains access control list configuration
type ACLConfig struct {
	Rules []ACLRuleConfig `json:"rules"`
}

// ACLRuleConfig represents a single ACL rule
type ACLRuleConfig struct {
	AgentID    string `json:"agent_id"`
	Resource   string `json:"resource"`
	Permission string `json:"permission"`
}

// ContextConfig contains context management configuration
type ContextConfig struct {
	WatchPaths     []string `json:"watch_paths"`
	IndexInterval  string   `json:"index_interval"`
	DefaultTTL     string   `json:"default_ttl"`     // TTL for context entries (e.g., "24h")
	PollInterval   string   `json:"poll_interval"`   // File polling interval (e.g., "5s")
	IgnorePatterns []string `json:"ignore_patterns"` // File patterns to ignore
}

// MCPConfig contains MCP integration configuration
type MCPConfig struct {
	Enabled      bool              `json:"enabled"`
	CacheTimeout string            `json:"cache_timeout"` // Resource cache timeout (e.g., "5m")
	Servers      []MCPServerConfig `json:"servers"`
}

// MCPServerConfig contains configuration for an MCP server connection
type MCPServerConfig struct {
	Name        string   `json:"name"`
	Transport   string   `json:"transport"` // "stdio" or "http"
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
	Env         []string `json:"env,omitempty"`
	BaseURL     string   `json:"base_url,omitempty"`
	AutoConnect bool     `json:"auto_connect"`
}

// TailscaleConfig contains Tailscale integration configuration
type TailscaleConfig struct {
	// Enabled enables Tailscale integration
	Enabled bool `json:"enabled"`
	// RequireAuth requires all requests to come from Tailscale network
	RequireAuth bool `json:"require_auth"`
	// AllowedTags is a list of Tailscale ACL tags that are allowed to connect
	AllowedTags []string `json:"allowed_tags"`
	// FallbackMode defines behavior when Tailscale is not available
	// "development" - allow localhost connections
	// "strict" - reject all non-Tailscale connections
	FallbackMode string `json:"fallback_mode"`
	// AutoRegisterAgents automatically registers agents from Tailscale nodes
	AutoRegisterAgents bool `json:"auto_register_agents"`
	// BindToTailscale binds the server only to the Tailscale interface
	BindToTailscale bool `json:"bind_to_tailscale"`
	// SocketPath is the path to the Tailscale daemon socket (optional)
	SocketPath string `json:"socket_path,omitempty"`
	// TagMappings maps Tailscale tags to AOI permissions
	TagMappings []TagMappingConfig `json:"tag_mappings,omitempty"`
}

// TagMappingConfig represents a mapping from Tailscale tag to AOI permission
type TagMappingConfig struct {
	Tag        string   `json:"tag"`
	Resources  []string `json:"resources"`
	Permission string   `json:"permission"` // "none", "read", "write", "admin"
}

// Load reads configuration from a JSON file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// LoadDefault returns a configuration with sensible defaults
func LoadDefault() *Config {
	return &Config{
		Agent: AgentConfig{
			ID:    "default-agent",
			Role:  "engineer",
			Owner: "system",
		},
		Network: NetworkConfig{
			ListenAddr: "0.0.0.0:8080",
			TLSEnabled: false,
		},
		ACL: ACLConfig{
			Rules: []ACLRuleConfig{},
		},
		Context: ContextConfig{
			WatchPaths:     []string{"."},
			IndexInterval:  "5m",
			DefaultTTL:     "24h",
			PollInterval:   "5s",
			IgnorePatterns: []string{".git", "node_modules", "__pycache__", "*.tmp"},
		},
		MCP: MCPConfig{
			Enabled:      false,
			CacheTimeout: "5m",
			Servers:      []MCPServerConfig{},
		},
		Tailscale: TailscaleConfig{
			Enabled:            false,
			RequireAuth:        false,
			AllowedTags:        []string{"tag:aoi-agent"},
			FallbackMode:       "development",
			AutoRegisterAgents: true,
			BindToTailscale:    false,
			TagMappings:        []TagMappingConfig{},
		},
	}
}

// ParsePermission converts a permission string to an integer level
func ParsePermission(perm string) int {
	switch perm {
	case "none":
		return 0
	case "read":
		return 1
	case "write":
		return 2
	case "admin":
		return 3
	default:
		return 0
	}
}
