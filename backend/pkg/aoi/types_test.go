package aoi

import (
	"encoding/json"
	"testing"
)

func TestAgentIdentity_JSON(t *testing.T) {
	agent := AgentIdentity{
		ID:           "test-agent",
		Role:         RoleEngineer,
		Owner:        "testuser",
		Capabilities: []string{"read", "write"},
		Endpoint:     "http://localhost:8080",
		Status:       "online",
	}

	// Test marshaling
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal agent: %v", err)
	}

	// Test unmarshaling
	var decoded AgentIdentity
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal agent: %v", err)
	}

	if decoded.ID != agent.ID {
		t.Errorf("Expected ID %s, got %s", agent.ID, decoded.ID)
	}
	if decoded.Role != agent.Role {
		t.Errorf("Expected Role %s, got %s", agent.Role, decoded.Role)
	}
}

func TestAgentIdentity_JSONWithMetadata(t *testing.T) {
	agent := AgentIdentity{
		ID:    "test-agent",
		Role:  RoleEngineer,
		Owner: "testuser",
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"region":  "us-west",
		},
		Status: "online",
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal agent: %v", err)
	}

	var decoded AgentIdentity
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal agent: %v", err)
	}

	if decoded.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %v", decoded.Metadata["version"])
	}
}

func TestAgentIdentity_AllRoles(t *testing.T) {
	roles := []AgentRole{RolePM, RoleEngineer, RoleQA, RoleDesign}

	for _, role := range roles {
		agent := AgentIdentity{
			ID:   "agent-" + string(role),
			Role: role,
		}

		data, err := json.Marshal(agent)
		if err != nil {
			t.Fatalf("Failed to marshal agent with role %s: %v", role, err)
		}

		var decoded AgentIdentity
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal agent with role %s: %v", role, err)
		}

		if decoded.Role != role {
			t.Errorf("Expected role %s, got %s", role, decoded.Role)
		}
	}
}

func TestAgentIdentity_EmptyFields(t *testing.T) {
	agent := AgentIdentity{}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal empty agent: %v", err)
	}

	var decoded AgentIdentity
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty agent: %v", err)
	}
}

func TestQuery_JSON(t *testing.T) {
	query := Query{
		ID:       "query-1",
		From:     "pm-tanaka",
		To:       "eng-suzuki",
		Query:    "What's the progress on authentication?",
		Priority: "normal",
		Async:    false,
	}

	data, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal query: %v", err)
	}

	var decoded Query
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	if decoded.Query != query.Query {
		t.Errorf("Expected Query %s, got %s", query.Query, decoded.Query)
	}
}

func TestQuery_JSONWithContextScope(t *testing.T) {
	query := Query{
		ID:           "query-1",
		From:         "pm-agent",
		To:           "eng-agent",
		Query:        "Status?",
		ContextScope: []string{"auth", "user-service"},
		Priority:     "high",
		Async:        true,
	}

	data, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal query: %v", err)
	}

	var decoded Query
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	if len(decoded.ContextScope) != 2 {
		t.Errorf("Expected 2 context scopes, got %d", len(decoded.ContextScope))
	}

	if decoded.Async != true {
		t.Errorf("Expected Async true, got %v", decoded.Async)
	}
}

func TestQuery_JSONWithMetadata(t *testing.T) {
	query := Query{
		ID:    "query-1",
		From:  "agent-1",
		To:    "agent-2",
		Query: "Test",
		Metadata: map[string]interface{}{
			"timestamp": "2024-01-01",
			"retries":   3,
		},
	}

	data, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal query: %v", err)
	}

	var decoded Query
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	if decoded.Metadata["retries"].(float64) != 3 {
		t.Errorf("Expected retries 3, got %v", decoded.Metadata["retries"])
	}
}

func TestQueryResult_JSON(t *testing.T) {
	result := QueryResult{
		Summary:   "Task completed successfully",
		Progress:  100,
		Blockers:  []string{"none"},
		Completed: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal query result: %v", err)
	}

	var decoded QueryResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal query result: %v", err)
	}

	if decoded.Summary != result.Summary {
		t.Errorf("Expected Summary %s, got %s", result.Summary, decoded.Summary)
	}

	if decoded.Progress != result.Progress {
		t.Errorf("Expected Progress %d, got %d", result.Progress, decoded.Progress)
	}
}

func TestQueryResult_JSONWithContextRefs(t *testing.T) {
	result := QueryResult{
		Summary:     "Analysis complete",
		ContextRefs: []string{"doc-1", "doc-2", "doc-3"},
		Completed:   true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal query result: %v", err)
	}

	var decoded QueryResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal query result: %v", err)
	}

	if len(decoded.ContextRefs) != 3 {
		t.Errorf("Expected 3 context refs, got %d", len(decoded.ContextRefs))
	}
}

func TestTask_JSON(t *testing.T) {
	task := Task{
		ID:   "task-1",
		Type: "deploy",
		Parameters: map[string]interface{}{
			"env":     "production",
			"version": "1.2.3",
		},
		Async:   true,
		Timeout: 300,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if decoded.Type != task.Type {
		t.Errorf("Expected Type %s, got %s", task.Type, decoded.Type)
	}

	if decoded.Parameters["env"] != "production" {
		t.Errorf("Expected env production, got %v", decoded.Parameters["env"])
	}
}

func TestTask_EmptyParameters(t *testing.T) {
	task := Task{
		ID:   "task-1",
		Type: "simple",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var decoded Task
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}
}

func TestTaskResult_JSON(t *testing.T) {
	result := TaskResult{
		TaskID: "task-1",
		Status: "completed",
		Output: "Deployment successful",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal task result: %v", err)
	}

	var decoded TaskResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal task result: %v", err)
	}

	if decoded.Status != result.Status {
		t.Errorf("Expected Status %s, got %s", result.Status, decoded.Status)
	}
}

func TestTaskResult_WithError(t *testing.T) {
	result := TaskResult{
		TaskID: "task-1",
		Status: "failed",
		Error:  "Connection timeout",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal task result: %v", err)
	}

	var decoded TaskResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal task result: %v", err)
	}

	if decoded.Error != result.Error {
		t.Errorf("Expected Error %s, got %s", result.Error, decoded.Error)
	}
}

func TestHealthResponse_JSON(t *testing.T) {
	health := HealthResponse{Status: "ok"}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal health: %v", err)
	}

	expected := `{"status":"ok"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestHealthResponse_DifferentStatuses(t *testing.T) {
	statuses := []string{"ok", "degraded", "error"}

	for _, status := range statuses {
		health := HealthResponse{Status: status}

		data, err := json.Marshal(health)
		if err != nil {
			t.Fatalf("Failed to marshal health with status %s: %v", status, err)
		}

		var decoded HealthResponse
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal health: %v", err)
		}

		if decoded.Status != status {
			t.Errorf("Expected status %s, got %s", status, decoded.Status)
		}
	}
}

func TestAgentRole_Constants(t *testing.T) {
	if RolePM != "pm" {
		t.Errorf("Expected RolePM to be 'pm', got %s", RolePM)
	}
	if RoleEngineer != "engineer" {
		t.Errorf("Expected RoleEngineer to be 'engineer', got %s", RoleEngineer)
	}
	if RoleQA != "qa" {
		t.Errorf("Expected RoleQA to be 'qa', got %s", RoleQA)
	}
	if RoleDesign != "design" {
		t.Errorf("Expected RoleDesign to be 'design', got %s", RoleDesign)
	}
}
