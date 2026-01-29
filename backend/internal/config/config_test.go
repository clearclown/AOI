package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	cfg := LoadDefault()
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Agent.ID == "" {
		t.Error("Expected non-empty agent ID")
	}

	if cfg.Agent.Role == "" {
		t.Error("Expected non-empty agent role")
	}

	if cfg.Network.ListenAddr == "" {
		t.Error("Expected non-empty listen address")
	}
}

func TestLoadDefault_AgentDefaults(t *testing.T) {
	cfg := LoadDefault()

	if cfg.Agent.ID != "default-agent" {
		t.Errorf("Expected default agent ID 'default-agent', got %s", cfg.Agent.ID)
	}

	if cfg.Agent.Role != "engineer" {
		t.Errorf("Expected default role 'engineer', got %s", cfg.Agent.Role)
	}

	if cfg.Agent.Owner != "system" {
		t.Errorf("Expected default owner 'system', got %s", cfg.Agent.Owner)
	}
}

func TestLoadDefault_NetworkDefaults(t *testing.T) {
	cfg := LoadDefault()

	if cfg.Network.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("Expected default listen address '0.0.0.0:8080', got %s", cfg.Network.ListenAddr)
	}

	if cfg.Network.TLSEnabled {
		t.Error("Expected TLS to be disabled by default")
	}
}

func TestLoadDefault_ACLDefaults(t *testing.T) {
	cfg := LoadDefault()

	if cfg.ACL.Rules == nil {
		t.Error("Expected ACL rules to be initialized")
	}

	if len(cfg.ACL.Rules) != 0 {
		t.Errorf("Expected empty ACL rules by default, got %d rules", len(cfg.ACL.Rules))
	}
}

func TestLoadDefault_ContextDefaults(t *testing.T) {
	cfg := LoadDefault()

	if len(cfg.Context.WatchPaths) == 0 {
		t.Error("Expected default watch paths to be set")
	}

	if cfg.Context.IndexInterval != "5m" {
		t.Errorf("Expected default index interval '5m', got %s", cfg.Context.IndexInterval)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"agent": {
			"id": "test-agent",
			"role": "pm",
			"owner": "test-owner"
		},
		"network": {
			"listen_addr": "127.0.0.1:9090",
			"tls_enabled": true
		},
		"acl": {
			"rules": [
				{
					"agent_id": "agent-1",
					"resource": "/api/query",
					"permission": "allow"
				}
			]
		},
		"context": {
			"watch_paths": ["/path1", "/path2"],
			"index_interval": "10m"
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Agent.ID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got %s", cfg.Agent.ID)
	}

	if cfg.Network.ListenAddr != "127.0.0.1:9090" {
		t.Errorf("Expected listen address '127.0.0.1:9090', got %s", cfg.Network.ListenAddr)
	}

	if !cfg.Network.TLSEnabled {
		t.Error("Expected TLS to be enabled")
	}

	if len(cfg.ACL.Rules) != 1 {
		t.Errorf("Expected 1 ACL rule, got %d", len(cfg.ACL.Rules))
	}

	if len(cfg.Context.WatchPaths) != 2 {
		t.Errorf("Expected 2 watch paths, got %d", len(cfg.Context.WatchPaths))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.json")

	// Only specify agent config, others should be zero values
	configJSON := `{
		"agent": {
			"id": "partial-agent",
			"role": "qa",
			"owner": "partial-owner"
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Agent.ID != "partial-agent" {
		t.Errorf("Expected agent ID 'partial-agent', got %s", cfg.Agent.ID)
	}

	// Network should have zero values
	if cfg.Network.ListenAddr != "" {
		t.Errorf("Expected empty listen address, got %s", cfg.Network.ListenAddr)
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.json")

	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestACLRuleConfig_Fields(t *testing.T) {
	rule := ACLRuleConfig{
		AgentID:    "agent-1",
		Resource:   "/api/query",
		Permission: "allow",
	}

	if rule.AgentID != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got %s", rule.AgentID)
	}

	if rule.Resource != "/api/query" {
		t.Errorf("Expected resource '/api/query', got %s", rule.Resource)
	}

	if rule.Permission != "allow" {
		t.Errorf("Expected permission 'allow', got %s", rule.Permission)
	}
}
