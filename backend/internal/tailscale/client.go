// Package tailscale provides Tailscale integration for secure closed-network communication.
package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// Common errors
var (
	ErrNotConnected     = errors.New("not connected to Tailscale")
	ErrNodeNotFound     = errors.New("node not found")
	ErrInvalidNodeID    = errors.New("invalid node ID")
	ErrConnectionFailed = errors.New("failed to connect to Tailscale")
)

// NodeInfo contains information about a Tailscale node
type NodeInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Hostname  string   `json:"hostname"`
	IPs       []string `json:"ips"`
	Online    bool     `json:"online"`
	Tags      []string `json:"tags"`
	ExitNode  bool     `json:"exitNode"`
	OS        string   `json:"os"`
	TailnetID string   `json:"tailnetId"`
	UserID    string   `json:"userId"`
	Created   string   `json:"created"`
	LastSeen  string   `json:"lastSeen"`
}

// Status represents the Tailscale daemon status
type Status struct {
	BackendState string               `json:"BackendState"`
	Self         *NodeInfo            `json:"Self"`
	Peer         map[string]*NodeInfo `json:"Peer"`
	TailscaleIPs []string             `json:"TailscaleIPs"`
	Health       []string             `json:"Health"`
}

// Client provides an interface to the Tailscale daemon
type Client interface {
	// GetStatus returns the current Tailscale status
	GetStatus(ctx context.Context) (*Status, error)
	// GetSelf returns the local node information
	GetSelf(ctx context.Context) (*NodeInfo, error)
	// GetPeer returns information about a specific peer by node ID
	GetPeer(ctx context.Context, nodeID string) (*NodeInfo, error)
	// GetPeerByIP returns information about a peer by IP address
	GetPeerByIP(ctx context.Context, ip string) (*NodeInfo, error)
	// GetPeers returns all connected peers
	GetPeers(ctx context.Context) ([]*NodeInfo, error)
	// IsConnected checks if we're connected to the Tailscale network
	IsConnected(ctx context.Context) bool
	// VerifyPeer verifies that a peer is part of the Tailscale network
	VerifyPeer(ctx context.Context, nodeID string) (bool, error)
	// GetNodeTags returns the ACL tags for a node
	GetNodeTags(ctx context.Context, nodeID string) ([]string, error)
	// IsTailscaleIP checks if an IP address is a Tailscale IP
	IsTailscaleIP(ip string) bool
}

// LocalClient implements Client using the local Tailscale daemon socket
type LocalClient struct {
	socketPath string
	httpClient *http.Client
	mu         sync.RWMutex
	statusCache *Status
	cacheTime   time.Time
	cacheTTL    time.Duration
}

// LocalClientOption is a functional option for configuring LocalClient
type LocalClientOption func(*LocalClient)

// WithSocketPath sets a custom socket path
func WithSocketPath(path string) LocalClientOption {
	return func(c *LocalClient) {
		c.socketPath = path
	}
}

// WithCacheTTL sets the cache TTL for status queries
func WithCacheTTL(ttl time.Duration) LocalClientOption {
	return func(c *LocalClient) {
		c.cacheTTL = ttl
	}
}

// NewLocalClient creates a new LocalClient that connects to the local Tailscale daemon
func NewLocalClient(opts ...LocalClientOption) (*LocalClient, error) {
	c := &LocalClient{
		socketPath: getDefaultSocketPath(),
		cacheTTL:   5 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Create HTTP client that connects via Unix socket
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", c.socketPath)
			},
		},
		Timeout: 10 * time.Second,
	}

	return c, nil
}

func getDefaultSocketPath() string {
	// Check common Tailscale socket paths
	paths := []string{
		"/var/run/tailscale/tailscaled.sock",
		"/run/tailscale/tailscaled.sock",
		"/tmp/tailscaled.sock",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return "/var/run/tailscale/tailscaled.sock"
}

// GetStatus returns the current Tailscale status
func (c *LocalClient) GetStatus(ctx context.Context) (*Status, error) {
	c.mu.RLock()
	if c.statusCache != nil && time.Since(c.cacheTime) < c.cacheTTL {
		status := c.statusCache
		c.mu.RUnlock()
		return status, nil
	}
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://local-tailscaled.sock/localapi/v0/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	c.mu.Lock()
	c.statusCache = &status
	c.cacheTime = time.Now()
	c.mu.Unlock()

	return &status, nil
}

// GetSelf returns the local node information
func (c *LocalClient) GetSelf(ctx context.Context) (*NodeInfo, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	if status.Self == nil {
		return nil, ErrNotConnected
	}

	return status.Self, nil
}

// GetPeer returns information about a specific peer by node ID
func (c *LocalClient) GetPeer(ctx context.Context, nodeID string) (*NodeInfo, error) {
	if nodeID == "" {
		return nil, ErrInvalidNodeID
	}

	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Check if it's ourselves
	if status.Self != nil && status.Self.ID == nodeID {
		return status.Self, nil
	}

	// Look in peers
	if peer, ok := status.Peer[nodeID]; ok {
		return peer, nil
	}

	return nil, ErrNodeNotFound
}

// GetPeerByIP returns information about a peer by IP address
func (c *LocalClient) GetPeerByIP(ctx context.Context, ip string) (*NodeInfo, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Check self IPs
	if status.Self != nil {
		for _, selfIP := range status.Self.IPs {
			if selfIP == ip {
				return status.Self, nil
			}
		}
	}

	// Check peer IPs
	for _, peer := range status.Peer {
		for _, peerIP := range peer.IPs {
			if peerIP == ip {
				return peer, nil
			}
		}
	}

	return nil, ErrNodeNotFound
}

// GetPeers returns all connected peers
func (c *LocalClient) GetPeers(ctx context.Context) ([]*NodeInfo, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]*NodeInfo, 0, len(status.Peer))
	for _, peer := range status.Peer {
		peers = append(peers, peer)
	}

	return peers, nil
}

// IsConnected checks if we're connected to the Tailscale network
func (c *LocalClient) IsConnected(ctx context.Context) bool {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false
	}

	return status.BackendState == "Running"
}

// VerifyPeer verifies that a peer is part of the Tailscale network
func (c *LocalClient) VerifyPeer(ctx context.Context, nodeID string) (bool, error) {
	peer, err := c.GetPeer(ctx, nodeID)
	if err != nil {
		if errors.Is(err, ErrNodeNotFound) {
			return false, nil
		}
		return false, err
	}

	return peer.Online, nil
}

// GetNodeTags returns the ACL tags for a node
func (c *LocalClient) GetNodeTags(ctx context.Context, nodeID string) ([]string, error) {
	peer, err := c.GetPeer(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return peer.Tags, nil
}

// IsTailscaleIP checks if an IP address is a Tailscale IP
func (c *LocalClient) IsTailscaleIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Tailscale uses 100.64.0.0/10 (CGNAT range) for IPv4
	_, cgnat, _ := net.ParseCIDR("100.64.0.0/10")
	if cgnat.Contains(parsedIP) {
		return true
	}

	// Tailscale uses fd7a:115c:a1e0::/48 for IPv6
	_, tsIPv6, _ := net.ParseCIDR("fd7a:115c:a1e0::/48")
	if tsIPv6.Contains(parsedIP) {
		return true
	}

	return false
}

// MockClient is a mock implementation of Client for testing
type MockClient struct {
	mu          sync.RWMutex
	self        *NodeInfo
	peers       map[string]*NodeInfo
	connected   bool
	shouldError error
}

// NewMockClient creates a new MockClient for testing
func NewMockClient() *MockClient {
	return &MockClient{
		peers:     make(map[string]*NodeInfo),
		connected: true,
	}
}

// SetSelf sets the mock self node
func (m *MockClient) SetSelf(node *NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.self = node
}

// AddPeer adds a mock peer
func (m *MockClient) AddPeer(node *NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[node.ID] = node
}

// SetConnected sets the mock connection state
func (m *MockClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// SetError sets an error to be returned by all operations
func (m *MockClient) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = err
}

func (m *MockClient) GetStatus(ctx context.Context) (*Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	state := "Stopped"
	if m.connected {
		state = "Running"
	}

	return &Status{
		BackendState: state,
		Self:         m.self,
		Peer:         m.peers,
	}, nil
}

func (m *MockClient) GetSelf(ctx context.Context) (*NodeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	if m.self == nil {
		return nil, ErrNotConnected
	}

	return m.self, nil
}

func (m *MockClient) GetPeer(ctx context.Context, nodeID string) (*NodeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	if nodeID == "" {
		return nil, ErrInvalidNodeID
	}

	if m.self != nil && m.self.ID == nodeID {
		return m.self, nil
	}

	if peer, ok := m.peers[nodeID]; ok {
		return peer, nil
	}

	return nil, ErrNodeNotFound
}

func (m *MockClient) GetPeerByIP(ctx context.Context, ip string) (*NodeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	if m.self != nil {
		for _, selfIP := range m.self.IPs {
			if selfIP == ip {
				return m.self, nil
			}
		}
	}

	for _, peer := range m.peers {
		for _, peerIP := range peer.IPs {
			if peerIP == ip {
				return peer, nil
			}
		}
	}

	return nil, ErrNodeNotFound
}

func (m *MockClient) GetPeers(ctx context.Context) ([]*NodeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	peers := make([]*NodeInfo, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}

	return peers, nil
}

func (m *MockClient) IsConnected(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return false
	}

	return m.connected
}

func (m *MockClient) VerifyPeer(ctx context.Context, nodeID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return false, m.shouldError
	}

	if m.self != nil && m.self.ID == nodeID {
		return m.self.Online, nil
	}

	if peer, ok := m.peers[nodeID]; ok {
		return peer.Online, nil
	}

	return false, nil
}

func (m *MockClient) GetNodeTags(ctx context.Context, nodeID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError != nil {
		return nil, m.shouldError
	}

	if m.self != nil && m.self.ID == nodeID {
		return m.self.Tags, nil
	}

	if peer, ok := m.peers[nodeID]; ok {
		return peer.Tags, nil
	}

	return nil, ErrNodeNotFound
}

func (m *MockClient) IsTailscaleIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	_, cgnat, _ := net.ParseCIDR("100.64.0.0/10")
	if cgnat.Contains(parsedIP) {
		return true
	}

	_, tsIPv6, _ := net.ParseCIDR("fd7a:115c:a1e0::/48")
	if tsIPv6.Contains(parsedIP) {
		return true
	}

	return false
}
