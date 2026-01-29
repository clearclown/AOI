package approval

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
	StatusExpired  ApprovalStatus = "expired"
)

// ApprovalRequest represents a Human-in-the-Loop approval request
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	Requester   string                 `json:"requester"`
	TaskType    string                 `json:"taskType"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params"`
	Status      ApprovalStatus         `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	ExpiresAt   time.Time              `json:"expiresAt"`
	ApprovedBy  string                 `json:"approvedBy,omitempty"`
	DeniedBy    string                 `json:"deniedBy,omitempty"`
	DenyReason  string                 `json:"denyReason,omitempty"`
}

// ApprovalManager manages HitL approval requests
type ApprovalManager struct {
	requests      map[string]*ApprovalRequest
	mu            sync.RWMutex
	defaultExpiry time.Duration
	callbacks     map[string]func(*ApprovalRequest)
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager() *ApprovalManager {
	am := &ApprovalManager{
		requests:      make(map[string]*ApprovalRequest),
		defaultExpiry: 24 * time.Hour, // Default 24 hour expiry
		callbacks:     make(map[string]func(*ApprovalRequest)),
	}
	// Start background cleanup
	go am.cleanupExpired()
	return am
}

// CreateRequest creates a new approval request
func (am *ApprovalManager) CreateRequest(requester, taskType, description string, params map[string]interface{}) (*ApprovalRequest, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	req := &ApprovalRequest{
		ID:          uuid.New().String(),
		Requester:   requester,
		TaskType:    taskType,
		Description: description,
		Params:      params,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(am.defaultExpiry),
	}

	am.requests[req.ID] = req
	return req, nil
}

// GetRequest retrieves an approval request by ID
func (am *ApprovalManager) GetRequest(id string) (*ApprovalRequest, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	req, ok := am.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	// Check for expiration
	if req.Status == StatusPending && time.Now().After(req.ExpiresAt) {
		req.Status = StatusExpired
		req.UpdatedAt = time.Now()
	}

	return req, nil
}

// ListPending returns all pending approval requests
func (am *ApprovalManager) ListPending() []*ApprovalRequest {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var pending []*ApprovalRequest
	now := time.Now()
	for _, req := range am.requests {
		if req.Status == StatusPending {
			if now.After(req.ExpiresAt) {
				req.Status = StatusExpired
				req.UpdatedAt = now
			} else {
				pending = append(pending, req)
			}
		}
	}
	return pending
}

// ListAll returns all approval requests with optional status filter
func (am *ApprovalManager) ListAll(statusFilter ApprovalStatus) []*ApprovalRequest {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var result []*ApprovalRequest
	for _, req := range am.requests {
		if statusFilter == "" || req.Status == statusFilter {
			result = append(result, req)
		}
	}
	return result
}

// Approve approves a request
func (am *ApprovalManager) Approve(id, approvedBy string) (*ApprovalRequest, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	req, ok := am.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != StatusPending {
		return nil, fmt.Errorf("request is not pending: current status is %s", req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = StatusExpired
		req.UpdatedAt = time.Now()
		return nil, fmt.Errorf("request has expired")
	}

	req.Status = StatusApproved
	req.ApprovedBy = approvedBy
	req.UpdatedAt = time.Now()

	// Trigger callback if registered
	if callback, ok := am.callbacks[id]; ok {
		go callback(req)
		delete(am.callbacks, id)
	}

	return req, nil
}

// Deny denies a request
func (am *ApprovalManager) Deny(id, deniedBy, reason string) (*ApprovalRequest, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	req, ok := am.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != StatusPending {
		return nil, fmt.Errorf("request is not pending: current status is %s", req.Status)
	}

	req.Status = StatusDenied
	req.DeniedBy = deniedBy
	req.DenyReason = reason
	req.UpdatedAt = time.Now()

	// Trigger callback if registered
	if callback, ok := am.callbacks[id]; ok {
		go callback(req)
		delete(am.callbacks, id)
	}

	return req, nil
}

// RegisterCallback registers a callback function to be called when a request is approved/denied
func (am *ApprovalManager) RegisterCallback(requestID string, callback func(*ApprovalRequest)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.callbacks[requestID] = callback
}

// cleanupExpired periodically marks expired requests
func (am *ApprovalManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		am.mu.Lock()
		now := time.Now()
		for _, req := range am.requests {
			if req.Status == StatusPending && now.After(req.ExpiresAt) {
				req.Status = StatusExpired
				req.UpdatedAt = now
			}
		}
		am.mu.Unlock()
	}
}

// HandleJSONRPC handles approval-related JSON-RPC methods
func (am *ApprovalManager) HandleJSONRPC(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "aoi.approval.create":
		return am.handleCreate(params)
	case "aoi.approval.get":
		return am.handleGet(params)
	case "aoi.approval.list":
		return am.handleList(params)
	case "aoi.approval.approve":
		return am.handleApprove(params)
	case "aoi.approval.deny":
		return am.handleDeny(params)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (am *ApprovalManager) handleCreate(params json.RawMessage) (interface{}, error) {
	var p struct {
		Requester   string                 `json:"requester"`
		TaskType    string                 `json:"taskType"`
		Description string                 `json:"description"`
		Params      map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return am.CreateRequest(p.Requester, p.TaskType, p.Description, p.Params)
}

func (am *ApprovalManager) handleGet(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return am.GetRequest(p.ID)
}

func (am *ApprovalManager) handleList(params json.RawMessage) (interface{}, error) {
	var p struct {
		Status string `json:"status,omitempty"`
	}
	if params != nil && len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	if p.Status == "" || p.Status == "pending" {
		return am.ListPending(), nil
	}
	return am.ListAll(ApprovalStatus(p.Status)), nil
}

func (am *ApprovalManager) handleApprove(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID         string `json:"id"`
		ApprovedBy string `json:"approvedBy"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return am.Approve(p.ID, p.ApprovedBy)
}

func (am *ApprovalManager) handleDeny(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID       string `json:"id"`
		DeniedBy string `json:"deniedBy"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return am.Deny(p.ID, p.DeniedBy, p.Reason)
}
