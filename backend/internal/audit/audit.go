package audit

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	EventQuery       AuditEventType = "query"
	EventExecute     AuditEventType = "execute"
	EventApproval    AuditEventType = "approval"
	EventNotify      AuditEventType = "notify"
	EventContextRead AuditEventType = "context_read"
	EventMCPCall     AuditEventType = "mcp_call"
	EventAgentJoin   AuditEventType = "agent_join"
	EventAgentLeave  AuditEventType = "agent_leave"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType AuditEventType         `json:"eventType"`
	FromAgent string                 `json:"fromAgent"`
	ToAgent   string                 `json:"toAgent"`
	Summary   string                 `json:"summary"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Success   bool                   `json:"success"`
	ErrorMsg  string                 `json:"errorMsg,omitempty"`
}

// AuditLogger manages audit log entries
type AuditLogger struct {
	entries    []*AuditEntry
	mu         sync.RWMutex
	maxEntries int
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		entries:    make([]*AuditEntry, 0),
		maxEntries: 10000, // Keep last 10000 entries
	}
}

// Log records an audit entry
func (al *AuditLogger) Log(eventType AuditEventType, fromAgent, toAgent, summary string, details map[string]interface{}, success bool, errorMsg string) *AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry := &AuditEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: eventType,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Summary:   summary,
		Details:   details,
		Success:   success,
		ErrorMsg:  errorMsg,
	}

	al.entries = append(al.entries, entry)

	// Trim if exceeds max
	if len(al.entries) > al.maxEntries {
		al.entries = al.entries[len(al.entries)-al.maxEntries:]
	}

	return entry
}

// Query represents audit log query parameters
type Query struct {
	FromAgent     string         `json:"fromAgent,omitempty"`
	ToAgent       string         `json:"toAgent,omitempty"`
	EventType     AuditEventType `json:"eventType,omitempty"`
	SearchTerm    string         `json:"searchTerm,omitempty"`
	StartTime     *time.Time     `json:"startTime,omitempty"`
	EndTime       *time.Time     `json:"endTime,omitempty"`
	SuccessOnly   *bool          `json:"successOnly,omitempty"`
	Limit         int            `json:"limit,omitempty"`
	Offset        int            `json:"offset,omitempty"`
	SortDescending bool          `json:"sortDescending,omitempty"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	Entries    []*AuditEntry `json:"entries"`
	TotalCount int           `json:"totalCount"`
	Offset     int           `json:"offset"`
	Limit      int           `json:"limit"`
}

// Search searches audit entries based on query parameters
func (al *AuditLogger) Search(q Query) *QueryResult {
	al.mu.RLock()
	defer al.mu.RUnlock()

	// Apply filters
	var filtered []*AuditEntry
	for _, entry := range al.entries {
		if q.FromAgent != "" && !strings.Contains(strings.ToLower(entry.FromAgent), strings.ToLower(q.FromAgent)) {
			continue
		}
		if q.ToAgent != "" && !strings.Contains(strings.ToLower(entry.ToAgent), strings.ToLower(q.ToAgent)) {
			continue
		}
		if q.EventType != "" && entry.EventType != q.EventType {
			continue
		}
		if q.SearchTerm != "" && !strings.Contains(strings.ToLower(entry.Summary), strings.ToLower(q.SearchTerm)) {
			continue
		}
		if q.StartTime != nil && entry.Timestamp.Before(*q.StartTime) {
			continue
		}
		if q.EndTime != nil && entry.Timestamp.After(*q.EndTime) {
			continue
		}
		if q.SuccessOnly != nil && entry.Success != *q.SuccessOnly {
			continue
		}
		filtered = append(filtered, entry)
	}

	totalCount := len(filtered)

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		if q.SortDescending {
			return filtered[i].Timestamp.After(filtered[j].Timestamp)
		}
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	// Apply pagination
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return &QueryResult{
		Entries:    filtered[start:end],
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	}
}

// GetRecent returns the most recent N entries
func (al *AuditLogger) GetRecent(count int) []*AuditEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if count <= 0 {
		count = 50
	}
	if count > len(al.entries) {
		count = len(al.entries)
	}

	// Return most recent
	start := len(al.entries) - count
	result := make([]*AuditEntry, count)
	copy(result, al.entries[start:])

	// Reverse to get newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetByID retrieves a specific audit entry by ID
func (al *AuditLogger) GetByID(id string) (*AuditEntry, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	for _, entry := range al.entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("audit entry not found: %s", id)
}

// GetStats returns audit statistics
func (al *AuditLogger) GetStats() map[string]interface{} {
	al.mu.RLock()
	defer al.mu.RUnlock()

	stats := map[string]interface{}{
		"totalEntries":  len(al.entries),
		"maxEntries":    al.maxEntries,
		"eventTypeCounts": make(map[string]int),
		"successCount":   0,
		"failureCount":   0,
	}

	eventCounts := stats["eventTypeCounts"].(map[string]int)
	for _, entry := range al.entries {
		eventCounts[string(entry.EventType)]++
		if entry.Success {
			stats["successCount"] = stats["successCount"].(int) + 1
		} else {
			stats["failureCount"] = stats["failureCount"].(int) + 1
		}
	}

	return stats
}

// HandleJSONRPC handles audit-related JSON-RPC methods
func (al *AuditLogger) HandleJSONRPC(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "aoi.audit.log":
		return al.handleLog(params)
	case "aoi.audit.get":
		return al.handleGet(params)
	case "aoi.audit.search":
		return al.handleSearch(params)
	case "aoi.audit.recent":
		return al.handleRecent(params)
	case "aoi.audit.stats":
		return al.GetStats(), nil
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (al *AuditLogger) handleLog(params json.RawMessage) (interface{}, error) {
	var p struct {
		EventType string                 `json:"eventType"`
		FromAgent string                 `json:"fromAgent"`
		ToAgent   string                 `json:"toAgent"`
		Summary   string                 `json:"summary"`
		Details   map[string]interface{} `json:"details"`
		Success   bool                   `json:"success"`
		ErrorMsg  string                 `json:"errorMsg"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return al.Log(AuditEventType(p.EventType), p.FromAgent, p.ToAgent, p.Summary, p.Details, p.Success, p.ErrorMsg), nil
}

func (al *AuditLogger) handleGet(params json.RawMessage) (interface{}, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	return al.GetByID(p.ID)
}

func (al *AuditLogger) handleSearch(params json.RawMessage) (interface{}, error) {
	var q Query
	if params != nil && len(params) > 0 {
		if err := json.Unmarshal(params, &q); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}
	return al.Search(q), nil
}

func (al *AuditLogger) handleRecent(params json.RawMessage) (interface{}, error) {
	var p struct {
		Count int `json:"count"`
	}
	if params != nil && len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}
	return al.GetRecent(p.Count), nil
}
