package protocol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aoi-protocol/aoi/internal/notify"
)

func TestWSHub_Creation(t *testing.T) {
	hub := NewWSHub(nil)

	if hub == nil {
		t.Fatal("Expected hub to be created")
	}

	if hub.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if hub.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}

	if hub.notifyMgr == nil {
		t.Error("Expected notify manager to be initialized")
	}
}

func TestWSHub_WithNotifyManager(t *testing.T) {
	notifyMgr := notify.NewNotificationManager()
	hub := NewWSHub(notifyMgr)

	if hub.notifyMgr != notifyMgr {
		t.Error("Expected provided notify manager to be used")
	}
}

func TestWSHub_GetClientCount_Empty(t *testing.T) {
	hub := NewWSHub(nil)

	if count := hub.GetClientCount(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestWSMessage_Serialization(t *testing.T) {
	msg := WSMessage{
		Type:      MessageTypeAgentUpdate,
		Timestamp: time.Now(),
		ID:        "test-id",
	}

	payload := AgentUpdatePayload{
		AgentID: "agent-1",
		Status:  "online",
		Role:    "engineer",
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg.Payload = payloadJSON

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded WSMessage
	if err := json.Unmarshal(msgJSON, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, decoded.Type)
	}

	if decoded.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, decoded.ID)
	}
}

func TestWSMessage_AllTypes(t *testing.T) {
	types := []string{
		MessageTypeAgentUpdate,
		MessageTypeAuditEntry,
		MessageTypeNotification,
		MessageTypeApprovalRequest,
		MessageTypePing,
		MessageTypePong,
		MessageTypeSubscribe,
		MessageTypeUnsubscribe,
		MessageTypeError,
	}

	for _, msgType := range types {
		msg := WSMessage{
			Type:      msgType,
			Timestamp: time.Now(),
		}

		_, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("Failed to marshal message type %s: %v", msgType, err)
		}
	}
}

func TestAgentUpdatePayload(t *testing.T) {
	payload := AgentUpdatePayload{
		AgentID:  "agent-1",
		Status:   "online",
		Role:     "engineer",
		Endpoint: "http://localhost:8080",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AgentUpdatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.AgentID != payload.AgentID {
		t.Errorf("Expected agent ID %s, got %s", payload.AgentID, decoded.AgentID)
	}

	if decoded.Status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, decoded.Status)
	}
}

func TestAuditEntryPayload(t *testing.T) {
	payload := AuditEntryPayload{
		ID:        "audit-1",
		From:      "agent-1",
		To:        "agent-2",
		EventType: "query",
		Summary:   "Test query",
		Metadata:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AuditEntryPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != payload.ID {
		t.Errorf("Expected ID %s, got %s", payload.ID, decoded.ID)
	}

	if decoded.EventType != payload.EventType {
		t.Errorf("Expected event type %s, got %s", payload.EventType, decoded.EventType)
	}
}

func TestApprovalRequestPayload(t *testing.T) {
	payload := ApprovalRequestPayload{
		ID:          "approval-1",
		Requester:   "agent-1",
		TaskType:    "code_review",
		Description: "Review PR #123",
		Status:      "pending",
		Params:      map[string]interface{}{"pr": 123},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ApprovalRequestPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != payload.ID {
		t.Errorf("Expected ID %s, got %s", payload.ID, decoded.ID)
	}

	if decoded.Status != payload.Status {
		t.Errorf("Expected status %s, got %s", payload.Status, decoded.Status)
	}
}

func TestSubscribePayload(t *testing.T) {
	payload := SubscribePayload{
		Topics: []string{MessageTypeAgentUpdate, MessageTypeAuditEntry},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SubscribePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(decoded.Topics))
	}
}

func TestWebSocketEndpoint_Upgrade(t *testing.T) {
	server := NewServer(nil, nil)

	// Create test server
	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	// Start the hub
	go server.wsHub.Run()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	// Connect via WebSocket
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected status 101, got %d", resp.StatusCode)
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Verify client count
	if count := server.wsHub.GetClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestWebSocketEndpoint_WithAgentID(t *testing.T) {
	server := NewServer(nil, nil)

	// Create test server
	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	// Start the hub
	go server.wsHub.Run()

	// Connect with agent ID
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws?agent_id=test-agent"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	if count := server.wsHub.GetClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
}

func TestWebSocketEndpoint_MultipleClients(t *testing.T) {
	server := NewServer(nil, nil)

	// Create test server
	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	// Start the hub
	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	// Connect multiple clients
	dialer := websocket.DefaultDialer
	var conns []*websocket.Conn

	for i := 0; i < 3; i++ {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial %d failed: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	if count := server.wsHub.GetClientCount(); count != 3 {
		t.Errorf("Expected 3 clients, got %d", count)
	}

	// Close all connections
	for _, conn := range conns {
		conn.Close()
	}
}

func TestWebSocket_PingPong(t *testing.T) {
	server := NewServer(nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Send ping message
	pingMsg := WSMessage{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read pong response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var pongMsg WSMessage
	if err := conn.ReadJSON(&pongMsg); err != nil {
		t.Fatalf("Failed to read pong: %v", err)
	}

	if pongMsg.Type != MessageTypePong {
		t.Errorf("Expected pong message, got %s", pongMsg.Type)
	}
}

func TestWebSocket_Subscribe(t *testing.T) {
	server := NewServer(nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Subscribe to specific topics
	subPayload := SubscribePayload{
		Topics: []string{MessageTypeAgentUpdate},
	}
	payloadJSON, _ := json.Marshal(subPayload)

	subMsg := WSMessage{
		Type:      MessageTypeSubscribe,
		Payload:   payloadJSON,
		Timestamp: time.Now(),
	}

	if err := conn.WriteJSON(subMsg); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// No explicit response expected for subscribe, just verify no error
}

func TestWSHub_BroadcastMessage(t *testing.T) {
	server := NewServer(nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	payload := AgentUpdatePayload{
		AgentID: "agent-1",
		Status:  "online",
	}

	if err := server.wsHub.BroadcastMessage(MessageTypeAgentUpdate, payload); err != nil {
		t.Fatalf("Failed to broadcast: %v", err)
	}

	// Read the broadcasted message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg WSMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read broadcast: %v", err)
	}

	if msg.Type != MessageTypeAgentUpdate {
		t.Errorf("Expected agent_update, got %s", msg.Type)
	}
}

func TestWSHub_BroadcastToTopic(t *testing.T) {
	server := NewServer(nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Wait for registration (default subscribed to all topics)
	time.Sleep(50 * time.Millisecond)

	// Broadcast to specific topic
	payload := AuditEntryPayload{
		ID:        "audit-1",
		EventType: "test",
	}

	if err := server.wsHub.BroadcastToTopic(MessageTypeAuditEntry, MessageTypeAuditEntry, payload); err != nil {
		t.Fatalf("Failed to broadcast to topic: %v", err)
	}

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg WSMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read broadcast: %v", err)
	}

	if msg.Type != MessageTypeAuditEntry {
		t.Errorf("Expected audit_entry, got %s", msg.Type)
	}
}

func TestWebSocket_ClientDisconnect(t *testing.T) {
	server := NewServer(nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	go server.wsHub.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/ws"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	if count := server.wsHub.GetClientCount(); count != 1 {
		t.Errorf("Expected 1 client before disconnect, got %d", count)
	}

	// Close the connection
	conn.Close()

	// Wait for unregistration
	time.Sleep(100 * time.Millisecond)

	if count := server.wsHub.GetClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", count)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	time.Sleep(time.Millisecond)
	id2 := generateID()

	if id1 == id2 {
		t.Error("Expected different IDs for different calls")
	}

	if id1 == "" || id2 == "" {
		t.Error("Expected non-empty IDs")
	}
}

func TestGetWSHub(t *testing.T) {
	server := NewServer(nil, nil)

	hub := server.GetWSHub()
	if hub == nil {
		t.Error("Expected GetWSHub to return hub")
	}

	if hub != server.wsHub {
		t.Error("Expected GetWSHub to return the same hub")
	}
}
