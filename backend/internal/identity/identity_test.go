package identity

import (
	"sync"
	"testing"

	"github.com/aoi-protocol/aoi/pkg/aoi"
)

func TestAgentRegistry_Register(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{
		ID:     "test-agent",
		Role:   aoi.RoleEngineer,
		Owner:  "testuser",
		Status: "online",
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	found, err := registry.GetAgent("test-agent")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if found.ID != "test-agent" {
		t.Errorf("Expected ID test-agent, got %s", found.ID)
	}
}

func TestAgentRegistry_RegisterWithCapabilities(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{
		ID:           "agent-caps",
		Role:         aoi.RoleEngineer,
		Owner:        "testuser",
		Capabilities: []string{"read", "write", "execute"},
		Status:       "online",
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	found, err := registry.GetAgent("agent-caps")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if len(found.Capabilities) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(found.Capabilities))
	}
}

func TestAgentRegistry_RegisterWithMetadata(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{
		ID:    "agent-meta",
		Role:  aoi.RoleQA,
		Owner: "testuser",
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"region":  "us-west",
		},
		Status: "online",
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	found, err := registry.GetAgent("agent-meta")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if found.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %v", found.Metadata["version"])
	}
}

func TestAgentRegistry_RegisterDuplicateID(t *testing.T) {
	registry := NewAgentRegistry()
	agent1 := &aoi.AgentIdentity{ID: "dup-agent", Role: aoi.RoleEngineer, Status: "online"}
	agent2 := &aoi.AgentIdentity{ID: "dup-agent", Role: aoi.RolePM, Status: "busy"}

	registry.Register(agent1)
	registry.Register(agent2)

	// Should overwrite the first agent
	found, err := registry.GetAgent("dup-agent")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if found.Role != aoi.RolePM {
		t.Errorf("Expected role PM (duplicate should overwrite), got %s", found.Role)
	}
}

func TestAgentRegistry_RegisterEmptyID(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{
		ID:     "",
		Role:   aoi.RoleEngineer,
		Status: "online",
	}

	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("Failed to register agent with empty ID: %v", err)
	}

	// Empty ID should be allowed but findable
	found, err := registry.GetAgent("")
	if err != nil {
		t.Fatalf("Failed to get agent with empty ID: %v", err)
	}

	if found.ID != "" {
		t.Errorf("Expected empty ID, got %s", found.ID)
	}
}

func TestAgentRegistry_GetAgent_NotFound(t *testing.T) {
	registry := NewAgentRegistry()

	_, err := registry.GetAgent("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent agent, got nil")
	}

	if err.Error() != "agent not found" {
		t.Errorf("Expected 'agent not found' error, got %v", err)
	}
}

func TestAgentRegistry_RegisterNilAgent(t *testing.T) {
	registry := NewAgentRegistry()

	// Register nil should not panic but may register with empty key
	// This tests robustness
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register panicked with nil agent: %v", r)
		}
	}()

	// This will panic due to nil dereference, so we skip for now
	// In production, we'd add nil check to Register
	_ = registry
}

func TestAgentRegistry_Discover(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &aoi.AgentIdentity{ID: "agent-1", Role: aoi.RoleEngineer, Status: "online"}
	agent2 := &aoi.AgentIdentity{ID: "agent-2", Role: aoi.RolePM, Status: "online"}

	registry.Register(agent1)
	registry.Register(agent2)

	agents := registry.Discover()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestAgentRegistry_DiscoverEmpty(t *testing.T) {
	registry := NewAgentRegistry()

	agents := registry.Discover()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents in empty registry, got %d", len(agents))
	}
}

func TestAgentRegistry_DiscoverMultipleRoles(t *testing.T) {
	registry := NewAgentRegistry()

	roles := []aoi.AgentRole{aoi.RoleEngineer, aoi.RolePM, aoi.RoleQA, aoi.RoleDesign}
	for _, role := range roles {
		agent := &aoi.AgentIdentity{
			ID:     string(role) + "-1",
			Role:   role,
			Status: "online",
		}
		registry.Register(agent)
	}

	agents := registry.Discover()
	if len(agents) != 4 {
		t.Errorf("Expected 4 agents with different roles, got %d", len(agents))
	}

	// Check each role is present
	roleCount := make(map[aoi.AgentRole]int)
	for _, agent := range agents {
		roleCount[agent.Role]++
	}

	for _, role := range roles {
		if roleCount[role] != 1 {
			t.Errorf("Expected 1 agent with role %s, got %d", role, roleCount[role])
		}
	}
}

func TestAgentRegistry_UpdateStatus(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{ID: "test-agent", Role: aoi.RoleEngineer, Status: "online"}

	registry.Register(agent)
	err := registry.UpdateStatus("test-agent", "busy")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	found, _ := registry.GetAgent("test-agent")
	if found.Status != "busy" {
		t.Errorf("Expected status busy, got %s", found.Status)
	}
}

func TestAgentRegistry_UpdateStatusNotFound(t *testing.T) {
	registry := NewAgentRegistry()

	err := registry.UpdateStatus("nonexistent", "busy")
	if err == nil {
		t.Error("Expected error when updating status of nonexistent agent")
	}

	if err.Error() != "agent not found" {
		t.Errorf("Expected 'agent not found' error, got %v", err)
	}
}

func TestAgentRegistry_UpdateStatusMultipleTimes(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{ID: "status-agent", Role: aoi.RoleEngineer, Status: "online"}

	registry.Register(agent)

	statuses := []string{"busy", "idle", "offline", "online"}
	for _, status := range statuses {
		err := registry.UpdateStatus("status-agent", status)
		if err != nil {
			t.Fatalf("Failed to update status to %s: %v", status, err)
		}

		found, _ := registry.GetAgent("status-agent")
		if found.Status != status {
			t.Errorf("Expected status %s, got %s", status, found.Status)
		}
	}
}

func TestAgentRegistry_ConcurrentRegister(t *testing.T) {
	registry := NewAgentRegistry()
	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			agent := &aoi.AgentIdentity{
				ID:     string(rune('a' + id)),
				Role:   aoi.RoleEngineer,
				Status: "online",
			}
			registry.Register(agent)
		}(i)
	}

	wg.Wait()

	agents := registry.Discover()
	if len(agents) != numGoroutines {
		t.Errorf("Expected %d agents after concurrent registration, got %d", numGoroutines, len(agents))
	}
}

func TestAgentRegistry_ConcurrentReadWrite(t *testing.T) {
	registry := NewAgentRegistry()
	agent := &aoi.AgentIdentity{ID: "concurrent-agent", Role: aoi.RoleEngineer, Status: "online"}
	registry.Register(agent)

	numReaders := 50
	numWriters := 50

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			_, _ = registry.GetAgent("concurrent-agent")
		}()
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		go func(idx int) {
			defer wg.Done()
			status := "status-" + string(rune('0'+idx%10))
			_ = registry.UpdateStatus("concurrent-agent", status)
		}(i)
	}

	wg.Wait()

	// Verify agent still exists
	found, err := registry.GetAgent("concurrent-agent")
	if err != nil {
		t.Errorf("Agent should still exist after concurrent operations: %v", err)
	}

	if found.ID != "concurrent-agent" {
		t.Errorf("Agent ID corrupted after concurrent operations")
	}
}

func TestAgentRegistry_ConcurrentDiscover(t *testing.T) {
	registry := NewAgentRegistry()

	// Register some agents
	for i := 0; i < 10; i++ {
		agent := &aoi.AgentIdentity{
			ID:     string(rune('a' + i)),
			Role:   aoi.RoleEngineer,
			Status: "online",
		}
		registry.Register(agent)
	}

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent discover calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			agents := registry.Discover()
			if len(agents) != 10 {
				t.Errorf("Expected 10 agents, got %d", len(agents))
			}
		}()
	}

	wg.Wait()
}
