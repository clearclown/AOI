package tailscale

import (
	"context"
	"testing"
)

func TestMockClient_GetStatus(t *testing.T) {
	client := NewMockClient()
	client.SetSelf(&NodeInfo{
		ID:       "node-1",
		Name:     "test-node",
		Hostname: "test-host",
		IPs:      []string{"100.64.0.1"},
		Online:   true,
		Tags:     []string{"tag:aoi-agent"},
	})

	ctx := context.Background()
	status, err := client.GetStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.BackendState != "Running" {
		t.Errorf("expected BackendState 'Running', got '%s'", status.BackendState)
	}

	if status.Self == nil {
		t.Error("expected Self to be set")
	}

	if status.Self.ID != "node-1" {
		t.Errorf("expected Self.ID 'node-1', got '%s'", status.Self.ID)
	}
}

func TestMockClient_GetSelf(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test without self set
	_, err := client.GetSelf(ctx)
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}

	// Test with self set
	expected := &NodeInfo{
		ID:       "node-1",
		Name:     "test-node",
		Hostname: "test-host",
		IPs:      []string{"100.64.0.1"},
		Online:   true,
	}
	client.SetSelf(expected)

	self, err := client.GetSelf(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if self.ID != expected.ID {
		t.Errorf("expected ID '%s', got '%s'", expected.ID, self.ID)
	}
}

func TestMockClient_GetPeer(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test with empty node ID
	_, err := client.GetPeer(ctx, "")
	if err != ErrInvalidNodeID {
		t.Errorf("expected ErrInvalidNodeID, got %v", err)
	}

	// Test with non-existent node
	_, err = client.GetPeer(ctx, "non-existent")
	if err != ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}

	// Add a peer and test retrieval
	peer := &NodeInfo{
		ID:       "peer-1",
		Name:     "peer-node",
		Hostname: "peer-host",
		IPs:      []string{"100.64.0.2"},
		Online:   true,
	}
	client.AddPeer(peer)

	found, err := client.GetPeer(ctx, "peer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found.ID != peer.ID {
		t.Errorf("expected ID '%s', got '%s'", peer.ID, found.ID)
	}
}

func TestMockClient_GetPeerByIP(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test with non-existent IP
	_, err := client.GetPeerByIP(ctx, "100.64.0.99")
	if err != ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}

	// Add self and peer with IPs
	client.SetSelf(&NodeInfo{
		ID:   "self-1",
		IPs:  []string{"100.64.0.1"},
		Name: "self",
	})
	client.AddPeer(&NodeInfo{
		ID:   "peer-1",
		IPs:  []string{"100.64.0.2", "100.64.0.3"},
		Name: "peer",
	})

	// Test finding self by IP
	node, err := client.GetPeerByIP(ctx, "100.64.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ID != "self-1" {
		t.Errorf("expected ID 'self-1', got '%s'", node.ID)
	}

	// Test finding peer by secondary IP
	node, err = client.GetPeerByIP(ctx, "100.64.0.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ID != "peer-1" {
		t.Errorf("expected ID 'peer-1', got '%s'", node.ID)
	}
}

func TestMockClient_GetPeers(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test with no peers
	peers, err := client.GetPeers(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}

	// Add peers
	client.AddPeer(&NodeInfo{ID: "peer-1"})
	client.AddPeer(&NodeInfo{ID: "peer-2"})

	peers, err = client.GetPeers(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}
}

func TestMockClient_IsConnected(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Default is connected
	if !client.IsConnected(ctx) {
		t.Error("expected IsConnected to return true by default")
	}

	// Set disconnected
	client.SetConnected(false)
	if client.IsConnected(ctx) {
		t.Error("expected IsConnected to return false after SetConnected(false)")
	}
}

func TestMockClient_VerifyPeer(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test with non-existent node
	verified, err := client.VerifyPeer(ctx, "non-existent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verified {
		t.Error("expected VerifyPeer to return false for non-existent node")
	}

	// Add online peer
	client.AddPeer(&NodeInfo{ID: "peer-1", Online: true})
	verified, err = client.VerifyPeer(ctx, "peer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !verified {
		t.Error("expected VerifyPeer to return true for online peer")
	}

	// Add offline peer
	client.AddPeer(&NodeInfo{ID: "peer-2", Online: false})
	verified, err = client.VerifyPeer(ctx, "peer-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verified {
		t.Error("expected VerifyPeer to return false for offline peer")
	}
}

func TestMockClient_GetNodeTags(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test with non-existent node
	_, err := client.GetNodeTags(ctx, "non-existent")
	if err != ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}

	// Add peer with tags
	client.AddPeer(&NodeInfo{
		ID:   "peer-1",
		Tags: []string{"tag:aoi-agent", "tag:production"},
	})

	tags, err := client.GetNodeTags(ctx, "peer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestMockClient_IsTailscaleIP(t *testing.T) {
	client := NewMockClient()

	tests := []struct {
		ip       string
		expected bool
	}{
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false},
		{"100.128.0.0", false},
		{"192.168.1.1", false},
		{"fd7a:115c:a1e0::1", true},
		{"2001:db8::1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		result := client.IsTailscaleIP(tt.ip)
		if result != tt.expected {
			t.Errorf("IsTailscaleIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestMockClient_SetError(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Set an error
	testErr := ErrConnectionFailed
	client.SetError(testErr)

	// All operations should return the error
	_, err := client.GetStatus(ctx)
	if err != testErr {
		t.Errorf("expected %v, got %v", testErr, err)
	}

	_, err = client.GetSelf(ctx)
	if err != testErr {
		t.Errorf("expected %v, got %v", testErr, err)
	}

	_, err = client.GetPeer(ctx, "node-1")
	if err != testErr {
		t.Errorf("expected %v, got %v", testErr, err)
	}

	if client.IsConnected(ctx) {
		t.Error("expected IsConnected to return false when error is set")
	}
}

func TestLocalClient_IsTailscaleIP(t *testing.T) {
	client, _ := NewLocalClient()

	tests := []struct {
		ip       string
		expected bool
	}{
		{"100.64.0.1", true},
		{"100.100.100.100", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false},
		{"100.128.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"fd7a:115c:a1e0::1", true},
		{"fd7a:115c:a1e0:ab12:4843:cd96:624c:3a06", true},
		{"2001:db8::1", false},
		{"invalid-ip", false},
		{"", false},
	}

	for _, tt := range tests {
		result := client.IsTailscaleIP(tt.ip)
		if result != tt.expected {
			t.Errorf("IsTailscaleIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestNewLocalClient_Options(t *testing.T) {
	// Test with custom socket path
	customPath := "/tmp/custom.sock"
	client, err := NewLocalClient(WithSocketPath(customPath))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.socketPath != customPath {
		t.Errorf("expected socketPath '%s', got '%s'", customPath, client.socketPath)
	}

	// Test with custom cache TTL
	client, err = NewLocalClient(WithCacheTTL(10 * 1000000000)) // 10 seconds in nanoseconds
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.cacheTTL != 10*1000000000 {
		t.Errorf("expected cacheTTL 10s, got %v", client.cacheTTL)
	}
}
