package audit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuditLogger_Log(t *testing.T) {
	al := NewAuditLogger()

	details := map[string]interface{}{
		"query":  "What is the status?",
		"method": "aoi.query",
	}

	entry := al.Log(EventQuery, "pm-agent", "eng-agent", "PM queried engineer for status", details, true, "")

	if entry.ID == "" {
		t.Error("Expected ID to be set")
	}
	if entry.FromAgent != "pm-agent" {
		t.Errorf("Expected fromAgent 'pm-agent', got '%s'", entry.FromAgent)
	}
	if entry.ToAgent != "eng-agent" {
		t.Errorf("Expected toAgent 'eng-agent', got '%s'", entry.ToAgent)
	}
	if entry.EventType != EventQuery {
		t.Errorf("Expected eventType 'query', got '%s'", entry.EventType)
	}
	if !entry.Success {
		t.Error("Expected success to be true")
	}
}

func TestAuditLogger_GetRecent(t *testing.T) {
	al := NewAuditLogger()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		al.Log(EventQuery, "agent", "agent", "Test entry", nil, true, "")
	}

	// Get recent 5
	recent := al.GetRecent(5)
	if len(recent) != 5 {
		t.Errorf("Expected 5 recent entries, got %d", len(recent))
	}

	// Verify newest first
	for i := 1; i < len(recent); i++ {
		if recent[i-1].Timestamp.Before(recent[i].Timestamp) {
			t.Error("Recent entries should be sorted newest first")
		}
	}
}

func TestAuditLogger_Search(t *testing.T) {
	al := NewAuditLogger()

	// Add various entries
	al.Log(EventQuery, "pm-agent", "eng-agent", "Status query", nil, true, "")
	al.Log(EventExecute, "pm-agent", "eng-agent", "Run tests", nil, true, "")
	al.Log(EventApproval, "eng-agent", "pm-agent", "Approval requested", nil, true, "")
	al.Log(EventQuery, "qa-agent", "eng-agent", "Test results query", nil, false, "timeout")

	// Search by event type
	result := al.Search(Query{EventType: EventQuery})
	if result.TotalCount != 2 {
		t.Errorf("Expected 2 query events, got %d", result.TotalCount)
	}

	// Search by from agent
	result = al.Search(Query{FromAgent: "pm"})
	if result.TotalCount != 2 {
		t.Errorf("Expected 2 entries from pm-agent, got %d", result.TotalCount)
	}

	// Search by success status
	successOnly := true
	result = al.Search(Query{SuccessOnly: &successOnly})
	if result.TotalCount != 3 {
		t.Errorf("Expected 3 successful entries, got %d", result.TotalCount)
	}

	// Search by search term
	result = al.Search(Query{SearchTerm: "test"})
	if result.TotalCount != 2 {
		t.Errorf("Expected 2 entries with 'test', got %d", result.TotalCount)
	}
}

func TestAuditLogger_SearchPagination(t *testing.T) {
	al := NewAuditLogger()

	// Add 25 entries
	for i := 0; i < 25; i++ {
		al.Log(EventQuery, "agent", "agent", "Entry", nil, true, "")
	}

	// Test pagination
	result := al.Search(Query{Limit: 10, Offset: 0})
	if len(result.Entries) != 10 {
		t.Errorf("Expected 10 entries, got %d", len(result.Entries))
	}
	if result.TotalCount != 25 {
		t.Errorf("Expected totalCount 25, got %d", result.TotalCount)
	}

	// Page 2
	result = al.Search(Query{Limit: 10, Offset: 10})
	if len(result.Entries) != 10 {
		t.Errorf("Expected 10 entries on page 2, got %d", len(result.Entries))
	}

	// Page 3 (partial)
	result = al.Search(Query{Limit: 10, Offset: 20})
	if len(result.Entries) != 5 {
		t.Errorf("Expected 5 entries on page 3, got %d", len(result.Entries))
	}
}

func TestAuditLogger_SearchTimeRange(t *testing.T) {
	al := NewAuditLogger()

	// Add entry
	entry := al.Log(EventQuery, "agent", "agent", "Test", nil, true, "")

	// Search with time range that includes entry
	start := entry.Timestamp.Add(-1 * time.Hour)
	end := entry.Timestamp.Add(1 * time.Hour)
	result := al.Search(Query{StartTime: &start, EndTime: &end})
	if result.TotalCount != 1 {
		t.Errorf("Expected 1 entry in time range, got %d", result.TotalCount)
	}

	// Search with time range that excludes entry
	future := entry.Timestamp.Add(1 * time.Hour)
	result = al.Search(Query{StartTime: &future})
	if result.TotalCount != 0 {
		t.Errorf("Expected 0 entries in future time range, got %d", result.TotalCount)
	}
}

func TestAuditLogger_GetByID(t *testing.T) {
	al := NewAuditLogger()

	entry := al.Log(EventQuery, "agent", "agent", "Test", nil, true, "")

	// Test retrieval
	retrieved, err := al.GetByID(entry.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.ID != entry.ID {
		t.Error("Retrieved wrong entry")
	}

	// Test not found
	_, err = al.GetByID("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent entry")
	}
}

func TestAuditLogger_GetStats(t *testing.T) {
	al := NewAuditLogger()

	al.Log(EventQuery, "a", "b", "Test 1", nil, true, "")
	al.Log(EventQuery, "a", "b", "Test 2", nil, false, "error")
	al.Log(EventExecute, "a", "b", "Test 3", nil, true, "")

	stats := al.GetStats()

	if stats["totalEntries"].(int) != 3 {
		t.Errorf("Expected 3 total entries, got %v", stats["totalEntries"])
	}
	if stats["successCount"].(int) != 2 {
		t.Errorf("Expected 2 success count, got %v", stats["successCount"])
	}
	if stats["failureCount"].(int) != 1 {
		t.Errorf("Expected 1 failure count, got %v", stats["failureCount"])
	}
}

func TestAuditLogger_MaxEntries(t *testing.T) {
	al := &AuditLogger{
		entries:    make([]*AuditEntry, 0),
		maxEntries: 5, // Small limit for testing
	}

	// Add 10 entries
	for i := 0; i < 10; i++ {
		al.Log(EventQuery, "agent", "agent", "Entry", nil, true, "")
	}

	if len(al.entries) != 5 {
		t.Errorf("Expected 5 entries after trimming, got %d", len(al.entries))
	}
}

func TestAuditLogger_JSONRPCHandlers(t *testing.T) {
	al := NewAuditLogger()

	// Test log
	logParams, _ := json.Marshal(map[string]interface{}{
		"eventType": "query",
		"fromAgent": "test-agent",
		"toAgent":   "target-agent",
		"summary":   "Test log entry",
		"success":   true,
	})

	result, err := al.HandleJSONRPC("aoi.audit.log", logParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC log failed: %v", err)
	}
	entry := result.(*AuditEntry)

	// Test get
	getParams, _ := json.Marshal(map[string]string{"id": entry.ID})
	result, err = al.HandleJSONRPC("aoi.audit.get", getParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC get failed: %v", err)
	}

	// Test search
	searchParams, _ := json.Marshal(map[string]string{"fromAgent": "test"})
	result, err = al.HandleJSONRPC("aoi.audit.search", searchParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC search failed: %v", err)
	}
	queryResult := result.(*QueryResult)
	if queryResult.TotalCount != 1 {
		t.Errorf("Expected 1 search result, got %d", queryResult.TotalCount)
	}

	// Test recent
	recentParams, _ := json.Marshal(map[string]int{"count": 10})
	result, err = al.HandleJSONRPC("aoi.audit.recent", recentParams)
	if err != nil {
		t.Fatalf("HandleJSONRPC recent failed: %v", err)
	}
	recent := result.([]*AuditEntry)
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent entry, got %d", len(recent))
	}

	// Test stats
	result, err = al.HandleJSONRPC("aoi.audit.stats", nil)
	if err != nil {
		t.Fatalf("HandleJSONRPC stats failed: %v", err)
	}
	stats := result.(map[string]interface{})
	if stats["totalEntries"].(int) != 1 {
		t.Errorf("Expected 1 total entry, got %v", stats["totalEntries"])
	}
}
