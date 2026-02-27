package h2a

import (
	"strings"
	"testing"
)

func TestH2AManager_RegisterSession(t *testing.T) {
	m := NewH2AManager()

	if err := m.RegisterSession("eng-suzuki", "claude-eng", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	session, ok := m.GetSession("eng-suzuki")
	if !ok {
		t.Fatal("expected session to be registered")
	}
	if session.AgentID != "eng-suzuki" {
		t.Errorf("agent_id: want 'eng-suzuki', got '%s'", session.AgentID)
	}
	if session.SessionName != "claude-eng" {
		t.Errorf("session_name: want 'claude-eng', got '%s'", session.SessionName)
	}
	if session.RegisteredAt.IsZero() {
		t.Error("registered_at should not be zero")
	}
}

func TestH2AManager_RegisterSession_InvalidArgs(t *testing.T) {
	m := NewH2AManager()

	if err := m.RegisterSession("", "session", ""); err == nil {
		t.Error("expected error for empty agent_id")
	}
	if err := m.RegisterSession("agent", "", ""); err == nil {
		t.Error("expected error for empty session_name")
	}
}

func TestH2AManager_ListSessions(t *testing.T) {
	m := NewH2AManager()
	_ = m.RegisterSession("eng-a", "session-a", "")
	_ = m.RegisterSession("eng-b", "session-b", "main")

	sessions := m.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestH2AManager_ListSessions_Empty(t *testing.T) {
	m := NewH2AManager()
	if sessions := m.ListSessions(); len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestH2AManager_CanSendTo_PM(t *testing.T) {
	m := NewH2AManager()
	m.SetPMUsers([]string{"pm-tanaka", "pm-yamada"})

	tests := []struct {
		fromUser        string
		targetAgentID   string
		expectAllowed   bool
	}{
		{"pm-tanaka", "eng-suzuki", true},
		{"pm-tanaka", "eng-yamada", true},
		{"pm-tanaka", "pm-tanaka", true},
		{"pm-yamada", "eng-suzuki", true},
	}

	for _, tt := range tests {
		got := m.CanSendTo(tt.fromUser, tt.targetAgentID)
		if got != tt.expectAllowed {
			t.Errorf("CanSendTo(%q, %q) = %v, want %v", tt.fromUser, tt.targetAgentID, got, tt.expectAllowed)
		}
	}
}

func TestH2AManager_CanSendTo_Engineer(t *testing.T) {
	m := NewH2AManager()
	m.SetPMUsers([]string{"pm-tanaka"})

	// engineers can only send to themselves
	if !m.CanSendTo("eng-suzuki", "eng-suzuki") {
		t.Error("engineer should be able to send to self")
	}
	if m.CanSendTo("eng-suzuki", "eng-yamada") {
		t.Error("engineer should not be able to send to others")
	}
}

func TestH2AManager_CanSendTo_EmptyArgs(t *testing.T) {
	m := NewH2AManager()
	if m.CanSendTo("", "eng-suzuki") {
		t.Error("empty fromUser should be denied")
	}
	if m.CanSendTo("user", "") {
		t.Error("empty targetAgentID should be denied")
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"valid simple command", "run tests", false},
		{"valid long but ok", strings.Repeat("a", MaxCommandLength), false},
		{"empty string", "", true},
		{"whitespace only", "   \t\n  ", true},
		{"too long", strings.Repeat("x", MaxCommandLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestH2AManager_SendCommand_NoSession(t *testing.T) {
	m := NewH2AManager()
	_, err := m.SendCommand("non-existent", "hello", false)
	if err == nil {
		t.Error("expected error when no session is registered")
	}
}

func TestH2AManager_SendCommand_InvalidCommand(t *testing.T) {
	m := NewH2AManager()
	_ = m.RegisterSession("eng-test", "test-session", "")

	_, err := m.SendCommand("eng-test", "", false)
	if err == nil {
		t.Error("expected error for empty command")
	}

	_, err = m.SendCommand("eng-test", strings.Repeat("z", MaxCommandLength+1), false)
	if err == nil {
		t.Error("expected error for too-long command")
	}
}

func TestH2AManager_CaptureOutput_NoSession(t *testing.T) {
	m := NewH2AManager()
	_, err := m.CaptureOutput("non-existent", 50)
	if err == nil {
		t.Error("expected error when no session is registered")
	}
}

func TestH2AManager_StopStream_NotFound(t *testing.T) {
	m := NewH2AManager()
	if err := m.StopStream("non-existent-stream"); err == nil {
		t.Error("expected error for unknown stream ID")
	}
}

func TestH2AManager_tmuxTarget(t *testing.T) {
	tests := []struct {
		session *TmuxSession
		want    string
	}{
		{&TmuxSession{SessionName: "sess"}, "sess"},
		{&TmuxSession{SessionName: "sess", PaneName: "main"}, "sess:main"},
	}
	for _, tt := range tests {
		got := tmuxTarget(tt.session)
		if got != tt.want {
			t.Errorf("tmuxTarget() = %q, want %q", got, tt.want)
		}
	}
}
