package protocol

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aoi-protocol/aoi/internal/notify"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// WebSocket message types
const (
	MessageTypeAgentUpdate     = "agent_update"
	MessageTypeAuditEntry      = "audit_entry"
	MessageTypeNotification    = "notification"
	MessageTypeApprovalRequest = "approval_request"
	MessageTypePing            = "ping"
	MessageTypePong            = "pong"
	MessageTypeSubscribe       = "subscribe"
	MessageTypeUnsubscribe     = "unsubscribe"
	MessageTypeError           = "error"
	// H2A: Human-to-Agent output streaming
	MessageTypeH2AOutput = "h2a_output"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	ID        string          `json:"id,omitempty"`
}

// AgentUpdatePayload represents an agent status update
type AgentUpdatePayload struct {
	AgentID  string `json:"agent_id"`
	Status   string `json:"status"`
	Role     string `json:"role,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// AuditEntryPayload represents an audit log entry
type AuditEntryPayload struct {
	ID        string            `json:"id"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	EventType string            `json:"event_type"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ApprovalRequestPayload represents an approval request
type ApprovalRequestPayload struct {
	ID          string                 `json:"id"`
	Requester   string                 `json:"requester"`
	TaskType    string                 `json:"task_type"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Status      string                 `json:"status"`
}

// SubscribePayload represents a subscription request
type SubscribePayload struct {
	Topics []string `json:"topics"`
}

// H2AOutputPayload is the WebSocket payload for Human-to-Agent output streaming.
type H2AOutputPayload struct {
	StreamID   string `json:"stream_id"`
	AgentID    string `json:"agent_id"`
	Output     string `json:"output"`
	IsComplete bool   `json:"is_complete"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development; restrict in production
		return true
	},
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	hub        *WSHub
	conn       *websocket.Conn
	send       chan []byte
	agentID    string
	topics     map[string]bool
	topicsMu   sync.RWMutex
	notifyChan chan notify.Notification
}

// WSHub manages all WebSocket connections
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	notifyMgr  *notify.NotificationManager
	mu         sync.RWMutex
}

// NewWSHub creates a new WebSocket hub
func NewWSHub(notifyMgr *notify.NotificationManager) *WSHub {
	if notifyMgr == nil {
		notifyMgr = notify.NewNotificationManager()
	}
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		notifyMgr:  notifyMgr,
	}
}

// Run starts the WebSocket hub
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.agentID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.notifyChan != nil && client.agentID != "" {
					h.notifyMgr.Unsubscribe(client.agentID, client.notifyChan)
				}
				log.Printf("WebSocket client disconnected: %s", client.agentID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, schedule for removal
					go func(c *WSClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (h *WSHub) BroadcastMessage(msgType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := WSMessage{
		Type:      msgType,
		Payload:   payloadJSON,
		Timestamp: time.Now(),
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- msgJSON
	return nil
}

// BroadcastToTopic sends a message to clients subscribed to a specific topic
func (h *WSHub) BroadcastToTopic(topic string, msgType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := WSMessage{
		Type:      msgType,
		Payload:   payloadJSON,
		Timestamp: time.Now(),
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.topicsMu.RLock()
		subscribed := client.topics[topic]
		client.topicsMu.RUnlock()

		if subscribed {
			select {
			case client.send <- msgJSON:
			default:
				// Client buffer full
			}
		}
	}

	return nil
}

// GetClientCount returns the number of connected clients
func (h *WSHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket handles WebSocket upgrade and connection
func (s *Server) HandleWebSocket(hub *WSHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		// Get agent ID from query param or header
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			agentID = r.Header.Get("X-Agent-ID")
		}
		if agentID == "" {
			agentID = "anonymous-" + generateID()
		}

		client := &WSClient{
			hub:     hub,
			conn:    conn,
			send:    make(chan []byte, 256),
			agentID: agentID,
			topics:  make(map[string]bool),
		}

		// Subscribe to all topics by default
		client.topics[MessageTypeAgentUpdate] = true
		client.topics[MessageTypeAuditEntry] = true
		client.topics[MessageTypeNotification] = true
		client.topics[MessageTypeApprovalRequest] = true

		// Subscribe to notification manager if agent ID is provided
		if agentID != "" && agentID[:10] != "anonymous-" {
			client.notifyChan = hub.notifyMgr.Subscribe(agentID)
			go client.forwardNotifications()
		}

		hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()
	}
}

// forwardNotifications forwards notifications from NotificationManager to WebSocket
func (c *WSClient) forwardNotifications() {
	for notif := range c.notifyChan {
		payload, _ := json.Marshal(notif)
		msg := WSMessage{
			Type:      MessageTypeNotification,
			Payload:   payload,
			Timestamp: time.Now(),
			ID:        notif.ID,
		}
		msgJSON, _ := json.Marshal(msg)

		select {
		case c.send <- msgJSON:
		default:
			// Channel full
		}
	}
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WSClient) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case MessageTypePing:
		c.sendPong()

	case MessageTypeSubscribe:
		var payload SubscribePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("Invalid subscribe payload")
			return
		}
		c.topicsMu.Lock()
		for _, topic := range payload.Topics {
			c.topics[topic] = true
		}
		c.topicsMu.Unlock()

	case MessageTypeUnsubscribe:
		var payload SubscribePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("Invalid unsubscribe payload")
			return
		}
		c.topicsMu.Lock()
		for _, topic := range payload.Topics {
			delete(c.topics, topic)
		}
		c.topicsMu.Unlock()
	}
}

// sendError sends an error message to the client
func (c *WSClient) sendError(message string) {
	payload, _ := json.Marshal(map[string]string{"message": message})
	msg := WSMessage{
		Type:      MessageTypeError,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	msgJSON, _ := json.Marshal(msg)

	select {
	case c.send <- msgJSON:
	default:
	}
}

// sendPong sends a pong response
func (c *WSClient) sendPong() {
	msg := WSMessage{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}
	msgJSON, _ := json.Marshal(msg)

	select {
	case c.send <- msgJSON:
	default:
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// generateID generates a simple unique ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
