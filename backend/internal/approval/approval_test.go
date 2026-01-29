package approval

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApprovalManager_CreateRequest(t *testing.T) {
	am := NewApprovalManager()

	params := map[string]interface{}{
		"command": "run tests",
		"scope":   "all",
	}

	req, err := am.CreateRequest("pm-agent", "execute_task", "Run all tests", params)
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	if req.ID == "" {
		t.Error("Expected ID to be set")
	}
	if req.Requester != "pm-agent" {
		t.Errorf("Expected requester 'pm-agent', got '%s'", req.Requester)
	}
	if req.TaskType != "execute_task" {
		t.Errorf("Expected taskType 'execute_task', got '%s'", req.TaskType)
	}
	if req.Status != StatusPending {
		t.Errorf("Expected status 'pending', got '%s'", req.Status)
	}
	if req.Params["command"] != "run tests" {
		t.Error("Params not preserved correctly")
	}
}

func TestApprovalManager_GetRequest(t *testing.T) {
	am := NewApprovalManager()

	req, _ := am.CreateRequest("pm-agent", "execute_task", "Test task", nil)

	// Test retrieval
	retrieved, err := am.GetRequest(req.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if retrieved.ID != req.ID {
		t.Error("Retrieved wrong request")
	}

	// Test not found
	_, err = am.GetRequest("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent request")
	}
}

func TestApprovalManager_ListPending(t *testing.T) {
	am := NewApprovalManager()

	// Create multiple requests
	am.CreateRequest("agent-1", "task-1", "Task 1", nil)
	am.CreateRequest("agent-2", "task-2", "Task 2", nil)
	am.CreateRequest("agent-3", "task-3", "Task 3", nil)

	pending := am.ListPending()
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending requests, got %d", len(pending))
	}
}

func TestApprovalManager_Approve(t *testing.T) {
	am := NewApprovalManager()

	req, _ := am.CreateRequest("pm-agent", "execute_task", "Test task", nil)

	// Approve the request
	approved, err := am.Approve(req.ID, "human-user")
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	if approved.Status != StatusApproved {
		t.Errorf("Expected status 'approved', got '%s'", approved.Status)
	}
	if approved.ApprovedBy != "human-user" {
		t.Errorf("Expected approvedBy 'human-user', got '%s'", approved.ApprovedBy)
	}

	// Verify pending list is updated
	pending := am.ListPending()
	if len(pending) != 0 {
		t.Error("Expected no pending requests after approval")
	}

	// Try to approve again
	_, err = am.Approve(req.ID, "another-user")
	if err == nil {
		t.Error("Expected error when approving non-pending request")
	}
}

func TestApprovalManager_Deny(t *testing.T) {
	am := NewApprovalManager()

	req, _ := am.CreateRequest("pm-agent", "execute_task", "Test task", nil)

	// Deny the request
	denied, err := am.Deny(req.ID, "human-user", "Not authorized")
	if err != nil {
		t.Fatalf("Deny failed: %v", err)
	}

	if denied.Status != StatusDenied {
		t.Errorf("Expected status 'denied', got '%s'", denied.Status)
	}
	if denied.DeniedBy != "human-user" {
		t.Errorf("Expected deniedBy 'human-user', got '%s'", denied.DeniedBy)
	}
	if denied.DenyReason != "Not authorized" {
		t.Errorf("Expected denyReason 'Not authorized', got '%s'", denied.DenyReason)
	}
}

func TestApprovalManager_Callback(t *testing.T) {
	am := NewApprovalManager()

	req, _ := am.CreateRequest("pm-agent", "execute_task", "Test task", nil)

	callbackCalled := make(chan bool, 1)
	am.RegisterCallback(req.ID, func(r *ApprovalRequest) {
		if r.Status == StatusApproved {
			callbackCalled <- true
		}
	})

	// Approve the request
	am.Approve(req.ID, "human-user")

	// Wait for callback
	select {
	case <-callbackCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Callback was not called within timeout")
	}
}

func TestApprovalManager_JSONRPCHandlers(t *testing.T) {
	am := NewApprovalManager()

	// Test create
	createParams, _ := json.Marshal(map[string]interface{}{
		"requester":   "test-agent",
		"taskType":    "test-task",
		"description": "Test description",
		"params":      map[string]interface{}{"key": "value"},
	})

	result, err := am.HandleJSONRPC("aoi.approval.create", createParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC create failed: %v", err)
	}
	req := result.(*ApprovalRequest)

	// Test get
	getParams, _ := json.Marshal(map[string]string{"id": req.ID})
	result, err = am.HandleJSONRPC("aoi.approval.get", getParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC get failed: %v", err)
	}

	// Test list
	result, err = am.HandleJSONRPC("aoi.approval.list", nil)
	if err != nil {
		t.Fatalf("HandleJSONRPC list failed: %v", err)
	}
	list := result.([]*ApprovalRequest)
	if len(list) != 1 {
		t.Errorf("Expected 1 pending request, got %d", len(list))
	}

	// Test approve
	approveParams, _ := json.Marshal(map[string]string{
		"id":         req.ID,
		"approvedBy": "test-user",
	})
	result, err = am.HandleJSONRPC("aoi.approval.approve", approveParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC approve failed: %v", err)
	}
	approved := result.(*ApprovalRequest)
	if approved.Status != StatusApproved {
		t.Errorf("Expected status approved, got %s", approved.Status)
	}
}

func TestApprovalManager_ListAll(t *testing.T) {
	am := NewApprovalManager()

	// Create and approve one request
	req1, _ := am.CreateRequest("agent-1", "task-1", "Task 1", nil)
	am.Approve(req1.ID, "user")

	// Create and deny another
	req2, _ := am.CreateRequest("agent-2", "task-2", "Task 2", nil)
	am.Deny(req2.ID, "user", "reason")

	// Create a pending one
	am.CreateRequest("agent-3", "task-3", "Task 3", nil)

	// List all
	all := am.ListAll("")
	if len(all) != 3 {
		t.Errorf("Expected 3 total requests, got %d", len(all))
	}

	// List only approved
	approved := am.ListAll(StatusApproved)
	if len(approved) != 1 {
		t.Errorf("Expected 1 approved request, got %d", len(approved))
	}

	// List only denied
	denied := am.ListAll(StatusDenied)
	if len(denied) != 1 {
		t.Errorf("Expected 1 denied request, got %d", len(denied))
	}
}
