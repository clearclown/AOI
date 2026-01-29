package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aoi-protocol/aoi/internal/acl"
	"github.com/aoi-protocol/aoi/internal/config"
	aoicontext "github.com/aoi-protocol/aoi/internal/context"
	agentidentity "github.com/aoi-protocol/aoi/internal/identity"
	"github.com/aoi-protocol/aoi/internal/mcp"
	"github.com/aoi-protocol/aoi/internal/notify"
	"github.com/aoi-protocol/aoi/internal/protocol"
	"github.com/aoi-protocol/aoi/internal/secretary"
	"github.com/aoi-protocol/aoi/internal/tailscale"
	"github.com/aoi-protocol/aoi/pkg/aoi"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "aoi.config.json", "Path to config file")
	addr := flag.String("addr", "", "Listen address (overrides config)")
	role := flag.String("role", "", "Agent role (overrides config)")
	agentID := flag.String("id", "", "Agent ID (overrides config)")
	owner := flag.String("owner", "", "Agent owner (overrides config)")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	if _, err := os.Stat(*configPath); err == nil {
		log.Printf("Loading config from %s", *configPath)
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Printf("Failed to load config: %v, using defaults", err)
			cfg = config.LoadDefault()
		}
	} else {
		log.Printf("Config file not found, using defaults")
		cfg = config.LoadDefault()
	}

	// Override config with command-line flags if provided
	if *addr != "" {
		cfg.Network.ListenAddr = *addr
	}
	if *agentID != "" {
		cfg.Agent.ID = *agentID
	}
	if *role != "" {
		cfg.Agent.Role = *role
	}
	if *owner != "" {
		cfg.Agent.Owner = *owner
	}

	// Map role string to AgentRole
	var agentRole aoi.AgentRole
	switch cfg.Agent.Role {
	case "pm":
		agentRole = aoi.RolePM
	case "engineer":
		agentRole = aoi.RoleEngineer
	case "qa":
		agentRole = aoi.RoleQA
	case "design":
		agentRole = aoi.RoleDesign
	default:
		agentRole = aoi.RoleEngineer
	}

	// Create agent identity
	identity := &aoi.AgentIdentity{
		ID:       cfg.Agent.ID,
		Role:     agentRole,
		Owner:    cfg.Agent.Owner,
		Status:   "online",
		Endpoint: fmt.Sprintf("http://%s", cfg.Network.ListenAddr),
	}

	// Create secretary
	sec := secretary.NewSecretary(identity)

	// Create registry
	registry := agentidentity.NewAgentRegistry()
	_ = registry.Register(identity)

	// Create ACL manager and configure rules
	aclMgr := acl.NewAclManager()
	for _, rule := range cfg.ACL.Rules {
		log.Printf("ACL Rule: %s -> %s: %s", rule.AgentID, rule.Resource, rule.Permission)
	}

	// Create notification manager
	notifyMgr := notify.NewNotificationManager()

	// Initialize Tailscale integration if enabled
	var tsIntegration *tailscale.Integration
	if cfg.Tailscale.Enabled {
		log.Printf("Initializing Tailscale integration...")
		var err error
		tsIntegration, err = tailscale.NewIntegration(cfg.Tailscale, registry, aclMgr)
		if err != nil {
			log.Printf("Failed to initialize Tailscale: %v", err)
			if cfg.Tailscale.FallbackMode != "development" {
				log.Fatal("Tailscale required but not available")
			}
		} else if tsIntegration != nil {
			log.Printf("   Tailscale: enabled")
			log.Printf("   RequireAuth: %v", cfg.Tailscale.RequireAuth)
			log.Printf("   AllowedTags: %v", cfg.Tailscale.AllowedTags)
			log.Printf("   FallbackMode: %s", cfg.Tailscale.FallbackMode)

			// Check Tailscale connection status
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if tsIntegration.Client.IsConnected(ctx) {
				selfNode, _ := tsIntegration.Client.GetSelf(ctx)
				if selfNode != nil {
					log.Printf("   Tailscale Node: %s (%s)", selfNode.Name, selfNode.ID)
					if len(selfNode.IPs) > 0 {
						log.Printf("   Tailscale IPs: %v", selfNode.IPs)
					}
					// Update identity with Tailscale node ID
					identity.TailscaleNodeID = selfNode.ID
				}
			} else {
				log.Printf("   Tailscale: not connected (running in fallback mode)")
			}
			cancel()
		}
	}

	// Initialize Context Monitor
	contextTTL := parseDuration(cfg.Context.DefaultTTL, 24*time.Hour)
	pollInterval := parseDuration(cfg.Context.PollInterval, 5*time.Second)

	contextStore := aoicontext.NewContextStore(contextTTL)
	contextMonitor := aoicontext.NewContextMonitor(contextStore)
	contextMonitor.SetPollInterval(pollInterval)
	contextAPI := aoicontext.NewContextAPI(contextMonitor, contextStore)

	// Add configured watch paths
	for _, watchPath := range cfg.Context.WatchPaths {
		if watchPath != "" && watchPath != "." {
			contextMonitor.AddWatch(aoicontext.WatchRequest{
				Path:         watchPath,
				Recursive:    true,
				IgnoreHidden: true,
			})
		}
	}

	// Start context monitor
	if err := contextMonitor.Start(); err != nil {
		log.Printf("Warning: Failed to start context monitor: %v", err)
	}

	// Initialize MCP Bridge
	mcpBridge := mcp.NewMCPBridge(contextStore)

	// Configure MCP servers if enabled
	if cfg.MCP.Enabled {
		log.Printf("Initializing MCP bridge...")
		cacheTimeout := parseDuration(cfg.MCP.CacheTimeout, 5*time.Minute)
		mcpBridge.Configure(&mcp.BridgeConfig{
			CacheTimeout: cacheTimeout,
		})

		for _, serverCfg := range cfg.MCP.Servers {
			if serverCfg.AutoConnect {
				clientConfig := &mcp.ClientConfig{
					ClientName:     "aoi-agent",
					ClientVersion:  "1.0.0",
					RequestTimeout: 60 * time.Second,
				}

				if serverCfg.Transport == "stdio" {
					clientConfig.Transport = mcp.TransportStdio
					clientConfig.Command = serverCfg.Command
					clientConfig.Args = serverCfg.Args
					clientConfig.Env = serverCfg.Env
				} else {
					clientConfig.Transport = mcp.TransportHTTP
					clientConfig.BaseURL = serverCfg.BaseURL
				}

				client := mcp.NewMCPClient(clientConfig)
				mcpBridge.AddClient(serverCfg.Name, client)
				log.Printf("   MCP Server: %s (%s)", serverCfg.Name, serverCfg.Transport)
			}
		}
	}

	// Create protocol server with JSON-RPC support
	server := protocol.NewServerWithContext(registry, aclMgr, contextAPI, mcpBridge)

	// Create HTTP mux for handlers
	mux := http.NewServeMux()

	// Set up legacy HTTP handlers for backward compatibility
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/query", queryHandler(sec))
	mux.HandleFunc("/api/agents", agentsHandler(registry))
	mux.HandleFunc("/api/status", statusHandler(sec))
	mux.HandleFunc("/api/notify", notifyHandler(notifyMgr))

	// Add Tailscale health endpoint if enabled
	if tsIntegration != nil {
		mux.HandleFunc("/tailscale/health", tailscaleHealthHandler(tsIntegration))
		mux.HandleFunc("/tailscale/peers", tailscalePeersHandler(tsIntegration))
	}

	// Wrap handlers with Tailscale auth middleware if enabled
	var handler http.Handler = mux
	if tsIntegration != nil && cfg.Tailscale.RequireAuth {
		handler = tsIntegration.Auth.Middleware(mux)
	}

	// Register the wrapped handler
	http.Handle("/", handler)

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start secretary in background
	go func() {
		if err := sec.Start(); err != nil {
			log.Printf("Secretary error: %v", err)
		}
	}()

	// Start HTTP server
	log.Printf("AOI Agent starting...")
	log.Printf("   ID: %s", identity.ID)
	log.Printf("   Role: %s", identity.Role)
	log.Printf("   Owner: %s", identity.Owner)
	log.Printf("   Listening on: %s", cfg.Network.ListenAddr)
	log.Printf("   TLS Enabled: %v", cfg.Network.TLSEnabled)
	log.Printf("   JSON-RPC 2.0: /api/v1/rpc")
	if cfg.Tailscale.Enabled {
		log.Printf("   Tailscale: enabled (require_auth=%v)", cfg.Tailscale.RequireAuth)
	}

	// Start protocol server (includes JSON-RPC endpoint)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start(cfg.Network.ListenAddr)
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case <-stop:
		log.Println("\nShutting down gracefully...")
		if err := sec.Shutdown(); err != nil {
			log.Printf("Secretary shutdown error: %v", err)
		}
		if err := contextMonitor.Stop(); err != nil {
			log.Printf("Context monitor shutdown error: %v", err)
		}
		contextStore.Stop()
		log.Println("Shutdown complete")
	}
}

// parseDuration parses a duration string, returning default if empty or invalid
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

// tailscaleHealthHandler returns Tailscale connection health
func tailscaleHealthHandler(ts *tailscale.Integration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		status := "unknown"
		nodeID := ""
		nodeName := ""
		var ips []string

		if ts.Client.IsConnected(ctx) {
			status = "connected"
			if self, err := ts.Client.GetSelf(ctx); err == nil {
				nodeID = self.ID
				nodeName = self.Name
				ips = self.IPs
			}
		} else {
			status = "disconnected"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    status,
			"node_id":   nodeID,
			"node_name": nodeName,
			"ips":       ips,
		})
	}
}

// tailscalePeersHandler returns list of Tailscale peers
func tailscalePeersHandler(ts *tailscale.Integration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		peers, err := ts.Client.GetPeers(ctx)
		if err != nil {
			http.Error(w, "Failed to get peers: "+err.Error(), http.StatusInternalServerError)
			return
		}

		peerList := make([]map[string]interface{}, 0, len(peers))
		for _, peer := range peers {
			peerList = append(peerList, map[string]interface{}{
				"id":       peer.ID,
				"name":     peer.Name,
				"hostname": peer.Hostname,
				"ips":      peer.IPs,
				"online":   peer.Online,
				"tags":     peer.Tags,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"peers": peerList,
			"count": len(peerList),
		})
	}
}

// healthHandler returns OK for health checks
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(aoi.HealthResponse{Status: "OK"})
}

// queryHandler processes query requests
func queryHandler(sec *secretary.Secretary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req secretary.QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		resp, err := sec.HandleQuery(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// agentsHandler returns list of registered agents
func agentsHandler(registry *agentidentity.AgentRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agents := registry.Discover()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		})
	}
}

// statusHandler returns secretary status
func statusHandler(sec *secretary.Secretary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   sec.GetStatus(),
			"identity": sec.Identity,
		})
	}
}

// notifyHandler handles notification requests
func notifyHandler(nm *notify.NotificationManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var notif notify.Notification
		if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := nm.Send(notif); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
		})
	}
}
