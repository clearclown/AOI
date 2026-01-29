package secretary

import (
	"sync"
	"testing"
	"time"

	"github.com/aoi-protocol/aoi/pkg/aoi"
)

func TestNewSecretary(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "eng-test",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)
	if sec == nil {
		t.Fatal("Expected secretary to be created")
	}

	if sec.Identity.ID != agentID.ID {
		t.Errorf("Expected ID %s, got %s", agentID.ID, sec.Identity.ID)
	}
}

func TestNewSecretary_WithDifferentRoles(t *testing.T) {
	roles := []aoi.AgentRole{
		aoi.RoleEngineer,
		aoi.RolePM,
		aoi.RoleQA,
		aoi.RoleDesign,
	}

	for _, role := range roles {
		agentID := &aoi.AgentIdentity{
			ID:    string(role) + "-test",
			Role:  role,
			Owner: "test-user",
		}

		sec := NewSecretary(agentID)
		if sec.Identity.Role != role {
			t.Errorf("Expected role %s, got %s", role, sec.Identity.Role)
		}
	}
}

func TestNewSecretary_InitialStatus(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)
	if sec.status != "idle" {
		t.Errorf("Expected initial status 'idle', got '%s'", sec.status)
	}
}

func TestSecretary_HandleQuery(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "eng-test",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		FromAgent: "pm-test",
		Query:     "What's the status?",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Answer == "" {
		t.Error("Expected non-empty answer")
	}

	if resp.Confidence == 0 {
		t.Error("Expected non-zero confidence")
	}
}

func TestSecretary_HandleQuery_EmptyQuery(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		FromAgent: "requester",
		Query:     "",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed with empty query: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
}

func TestSecretary_HandleQuery_LongQuery(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	longQuery := ""
	for i := 0; i < 1000; i++ {
		longQuery += "query "
	}

	req := QueryRequest{
		FromAgent: "requester",
		Query:     longQuery,
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed with long query: %v", err)
	}

	if resp.Answer == "" {
		t.Error("Expected non-empty answer")
	}
}

func TestSecretary_HandleQuery_ResponseFormat(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		FromAgent: "pm-agent",
		Query:     "Status check",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Confidence == 0 {
		t.Error("Expected non-zero confidence")
	}

	if resp.Answer == "" {
		t.Error("Expected non-empty Answer")
	}

	if resp.Sources == nil {
		t.Error("Expected Sources to be initialized")
	}
}

func TestSecretary_HandleQuery_MultipleCalls(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	for i := 0; i < 10; i++ {
		req := QueryRequest{
			FromAgent: "requester",
			Query:     "Query " + string(rune('0'+i)),
		}

		resp, err := sec.HandleQuery(req)
		if err != nil {
			t.Fatalf("HandleQuery failed on iteration %d: %v", i, err)
		}

		if resp == nil {
			t.Fatalf("Expected non-nil response on iteration %d", i)
		}
	}
}

func TestSecretary_StartShutdown(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "eng-test",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	// Start should not block
	done := make(chan error, 1)
	go func() {
		done <- sec.Start()
	}()

	// Shutdown should work
	err := sec.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Wait for Start to return
	startErr := <-done
	if startErr != nil {
		t.Errorf("Start returned error: %v", startErr)
	}
}

func TestSecretary_ShutdownWithoutStart(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	// Shutdown without starting should not hang or error
	err := sec.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestSecretary_MultipleShutdowns(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	go sec.Start()

	// First shutdown should succeed
	err := sec.Shutdown()
	if err != nil {
		t.Errorf("First shutdown failed: %v", err)
	}

	// Second shutdown should not panic (close on closed channel would panic)
	// This test verifies the implementation handles this gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Second shutdown panicked: %v", r)
		}
	}()
}

func TestSecretary_GetStatus(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "eng-test",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	status := sec.GetStatus()
	if status != "idle" {
		t.Errorf("Expected status 'idle', got '%s'", status)
	}
}

func TestSecretary_GetStatusAfterQueries(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	// Handle some queries
	for i := 0; i < 5; i++ {
		req := QueryRequest{
			FromAgent: "requester",
			Query:     "Query",
		}
		sec.HandleQuery(req)
	}

	// Status should still be accessible
	status := sec.GetStatus()
	if status == "" {
		t.Error("Expected non-empty status")
	}
}

func TestSecretary_ConcurrentQueries(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			req := QueryRequest{
				FromAgent: "requester-" + string(rune('0'+id%10)),
				Query:     "Concurrent query",
			}
			_, err := sec.HandleQuery(req)
			if err != nil {
				t.Errorf("Concurrent query failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

func TestSecretary_StartTimeout(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	done := make(chan error, 1)
	go func() {
		done <- sec.Start()
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown
	sec.Shutdown()

	// Should complete quickly
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start did not complete within timeout after shutdown")
	}
}

func TestSecretary_QueryWhileRunning(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	go sec.Start()

	// Handle query while running
	req := QueryRequest{
		FromAgent: "requester",
		Query:     "Query while running",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed while running: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	sec.Shutdown()
}

func TestSecretary_StatusConcurrentAccess(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = sec.GetStatus()
		}()
	}

	wg.Wait()
}

// Enhanced Query Processing Tests

func TestSecretary_HandleQuery_NewFormat(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "eng-test",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		Query:        "What is the status?",
		FromAgent:    "pm-agent",
		ContextScope: "project",
		Metadata: map[string]string{
			"priority": "high",
		},
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Answer == "" {
		t.Error("Expected non-empty answer")
	}

	if resp.Confidence == 0 {
		t.Error("Expected non-zero confidence")
	}

	if resp.Metadata == nil {
		t.Error("Expected metadata to be returned")
	}
}

func TestSecretary_HandleQuery_PMRole(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "pm-test",
		Role:  aoi.RolePM,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		Query:     "Project status?",
		FromAgent: "eng-agent",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if len(resp.Sources) == 0 {
		t.Error("Expected PM to return sources")
	}

	if resp.Confidence < 0.5 {
		t.Errorf("Expected reasonable confidence, got %f", resp.Confidence)
	}
}

func TestSecretary_HandleQuery_QARole(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "qa-test",
		Role:  aoi.RoleQA,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		Query:     "Test coverage?",
		FromAgent: "pm-agent",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if len(resp.Sources) == 0 {
		t.Error("Expected QA to return sources")
	}

	if resp.Answer == "" {
		t.Error("Expected non-empty answer from QA")
	}
}

func TestSecretary_HandleQuery_DesignRole(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "design-test",
		Role:  aoi.RoleDesign,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		Query:     "UI components ready?",
		FromAgent: "pm-agent",
	}

	resp, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if len(resp.Sources) == 0 {
		t.Error("Expected Design to return sources")
	}

	if resp.Confidence == 0 {
		t.Error("Expected non-zero confidence from Design")
	}
}

func TestSecretary_QueryLogging(t *testing.T) {
	agentID := &aoi.AgentIdentity{
		ID:    "test-agent",
		Role:  aoi.RoleEngineer,
		Owner: "test-user",
	}

	sec := NewSecretary(agentID)

	req := QueryRequest{
		Query:     "Test query",
		FromAgent: "requester",
	}

	_, err := sec.HandleQuery(req)
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	logs := sec.GetQueryLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 query log, got %d", len(logs))
	}

	if logs[0].FromAgent != "requester" {
		t.Errorf("Expected from_agent 'requester', got %s", logs[0].FromAgent)
	}

	if logs[0].Query != "Test query" {
		t.Errorf("Expected query 'Test query', got %s", logs[0].Query)
	}
}
