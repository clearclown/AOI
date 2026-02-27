package h2a

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MaxCommandLength is the maximum length of a command that can be sent to an agent.
const MaxCommandLength = 4096

// WSHubInterface allows H2AManager to broadcast output without importing the protocol package.
type WSHubInterface interface {
	BroadcastToTopic(topic string, msgType string, payload interface{}) error
}

// TmuxSession holds the tmux session information for an agent.
type TmuxSession struct {
	AgentID      string    `json:"agent_id"`
	SessionName  string    `json:"session_name"`
	PaneName     string    `json:"pane_name,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

// SendResult is the result of an aoi.h2a.send call.
type SendResult struct {
	Status   string `json:"status"`
	Output   string `json:"output,omitempty"`
	StreamID string `json:"stream_id,omitempty"`
}

// streamSession is an internal handle for a running stream goroutine.
type streamSession struct {
	id      string
	agentID string
	cancel  func()
}

// H2AManager manages Human-to-Agent interactions via tmux.
type H2AManager struct {
	sessions map[string]*TmuxSession   // agentID -> session
	streams  map[string]*streamSession // streamID -> stream
	pmUsers  []string                  // user IDs that hold PM (admin) permission
	wsHub    WSHubInterface
	mu       sync.RWMutex
}

// NewH2AManager creates a new H2AManager.
func NewH2AManager() *H2AManager {
	return &H2AManager{
		sessions: make(map[string]*TmuxSession),
		streams:  make(map[string]*streamSession),
		pmUsers:  []string{},
	}
}

// SetWSHub injects the WebSocket hub used to broadcast captured output.
func (m *H2AManager) SetWSHub(hub WSHubInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsHub = hub
}

// SetPMUsers configures which user IDs are treated as PM (can send to any agent).
func (m *H2AManager) SetPMUsers(users []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pmUsers = users
}

// RegisterSession associates a tmux session with an agent ID.
func (m *H2AManager) RegisterSession(agentID, sessionName, paneName string) error {
	if agentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if sessionName == "" {
		return fmt.Errorf("session_name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[agentID] = &TmuxSession{
		AgentID:      agentID,
		SessionName:  sessionName,
		PaneName:     paneName,
		RegisteredAt: time.Now(),
	}
	return nil
}

// GetSession returns the registered TmuxSession for agentID.
func (m *H2AManager) GetSession(agentID string) (*TmuxSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[agentID]
	return s, ok
}

// ListSessions returns all registered sessions.
func (m *H2AManager) ListSessions() []*TmuxSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*TmuxSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result
}

// CanSendTo returns true if fromUser is allowed to send commands to targetAgentID.
// PM users can send to any agent; all others can only send to themselves.
func (m *H2AManager) CanSendTo(fromUser, targetAgentID string) bool {
	if fromUser == "" || targetAgentID == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, pm := range m.pmUsers {
		if pm == fromUser {
			return true
		}
	}
	return fromUser == targetAgentID
}

// ValidateCommand checks that a command is safe to forward to tmux.
func ValidateCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}
	if len(command) > MaxCommandLength {
		return fmt.Errorf("command too long (%d bytes, max %d)", len(command), MaxCommandLength)
	}
	return nil
}

// tmuxTarget builds the -t argument for tmux commands.
func tmuxTarget(s *TmuxSession) string {
	if s.PaneName != "" {
		return s.SessionName + ":" + s.PaneName
	}
	return s.SessionName
}

// SendCommand sends command to the agent's tmux session.
// If captureOutput is true, it waits briefly then captures the pane.
func (m *H2AManager) SendCommand(agentID, command string, captureOutput bool) (*SendResult, error) {
	if err := ValidateCommand(command); err != nil {
		return nil, err
	}

	session, ok := m.GetSession(agentID)
	if !ok {
		return nil, fmt.Errorf("no tmux session registered for agent: %s", agentID)
	}

	target := tmuxTarget(session)
	if err := exec.Command("tmux", "send-keys", "-t", target, command, "Enter").Run(); err != nil {
		return nil, fmt.Errorf("tmux send-keys failed: %w", err)
	}

	result := &SendResult{Status: "sent"}

	if captureOutput {
		time.Sleep(300 * time.Millisecond)
		out, err := m.CaptureOutput(agentID, 50)
		if err == nil {
			result.Output = out
		}
	}

	return result, nil
}

// CaptureOutput runs tmux capture-pane and returns the last `lines` lines of output.
func (m *H2AManager) CaptureOutput(agentID string, lines int) (string, error) {
	session, ok := m.GetSession(agentID)
	if !ok {
		return "", fmt.Errorf("no tmux session registered for agent: %s", agentID)
	}

	if lines <= 0 {
		lines = 50
	}

	target := tmuxTarget(session)
	args := []string{
		"capture-pane", "-t", target,
		"-p",
		"-S", fmt.Sprintf("-%d", lines),
	}

	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane failed: %w", err)
	}
	return string(out), nil
}

// StartStream begins periodic output capture for agentID and broadcasts over WebSocket.
// Returns a stream ID that can be used to stop the stream.
func (m *H2AManager) StartStream(agentID string, intervalMs int) (string, error) {
	if _, ok := m.GetSession(agentID); !ok {
		return "", fmt.Errorf("no tmux session registered for agent: %s", agentID)
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}

	streamID := uuid.New().String()
	done := make(chan struct{})
	ss := &streamSession{
		id:      streamID,
		agentID: agentID,
		cancel:  sync.OnceFunc(func() { close(done) }),
	}

	m.mu.Lock()
	m.streams[streamID] = ss
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.streams, streamID)
			m.mu.Unlock()
		}()

		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		defer ticker.Stop()

		var lastOutput string
		for {
			select {
			case <-done:
				m.broadcastH2AOutput(streamID, agentID, "", true)
				return
			case <-ticker.C:
				output, err := m.CaptureOutput(agentID, 50)
				if err != nil || output == lastOutput {
					continue
				}
				lastOutput = output
				m.broadcastH2AOutput(streamID, agentID, output, false)
			}
		}
	}()

	return streamID, nil
}

// StopStream cancels a running stream.
func (m *H2AManager) StopStream(streamID string) error {
	m.mu.Lock()
	ss, ok := m.streams[streamID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("stream not found: %s", streamID)
	}
	ss.cancel()
	return nil
}

func (m *H2AManager) broadcastH2AOutput(streamID, agentID, output string, isComplete bool) {
	m.mu.RLock()
	hub := m.wsHub
	m.mu.RUnlock()
	if hub == nil {
		return
	}
	_ = hub.BroadcastToTopic("h2a:"+agentID, "h2a_output", map[string]interface{}{
		"stream_id":   streamID,
		"agent_id":    agentID,
		"output":      output,
		"is_complete": isComplete,
	})
}
