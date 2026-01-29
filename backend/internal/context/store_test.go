package context

import (
	"testing"
	"time"
)

func TestContextStore_Store(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	entry := &ContextEntry{
		Type:    ContextTypeFile,
		Source:  "test",
		Content: "test content",
		Summary: "test summary",
		Project: "test-project",
		File:    "/path/to/file.go",
		Topics:  []string{"golang", "testing"},
	}

	err := store.Store(entry)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if entry.ID == "" {
		t.Error("Expected entry ID to be set")
	}
	if entry.Timestamp.IsZero() {
		t.Error("Expected entry timestamp to be set")
	}
	if entry.ExpiresAt.IsZero() {
		t.Error("Expected entry expiration to be set")
	}
}

func TestContextStore_Get(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	entry := &ContextEntry{
		Type:    ContextTypeFile,
		Source:  "test",
		Content: "test content",
	}

	store.Store(entry)

	retrieved, err := store.Get(entry.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != entry.ID {
		t.Errorf("Expected ID %s, got %s", entry.ID, retrieved.ID)
	}
	if retrieved.Content != entry.Content {
		t.Errorf("Expected content %s, got %s", entry.Content, retrieved.Content)
	}
}

func TestContextStore_GetNotFound(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	_, err := store.Get("nonexistent")
	if err != ErrEntryNotFound {
		t.Errorf("Expected ErrEntryNotFound, got %v", err)
	}
}

func TestContextStore_Query_ByProject(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	// Store entries for different projects
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-a", Content: "content 1"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-a", Content: "content 2"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-b", Content: "content 3"})

	history, err := store.Query(ContextQuery{Project: "project-a"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}

	for _, entry := range history.Entries {
		if entry.Project != "project-a" {
			t.Errorf("Expected project project-a, got %s", entry.Project)
		}
	}
}

func TestContextStore_Query_ByFile(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	store.Store(&ContextEntry{Type: ContextTypeFile, File: "/path/to/file1.go", Content: "content 1"})
	store.Store(&ContextEntry{Type: ContextTypeFile, File: "/path/to/file1.go", Content: "content 2"})
	store.Store(&ContextEntry{Type: ContextTypeFile, File: "/path/to/file2.go", Content: "content 3"})

	history, err := store.Query(ContextQuery{File: "/path/to/file1.go"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
}

func TestContextStore_Query_ByTopic(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	store.Store(&ContextEntry{Type: ContextTypeFile, Topics: []string{"golang", "testing"}, Content: "content 1"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Topics: []string{"golang", "backend"}, Content: "content 2"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Topics: []string{"python"}, Content: "content 3"})

	history, err := store.Query(ContextQuery{Topic: "golang"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
}

func TestContextStore_Query_ByType(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	store.Store(&ContextEntry{Type: ContextTypeFile, Content: "file content"})
	store.Store(&ContextEntry{Type: ContextTypeActivity, Content: "activity content"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Content: "another file"})

	history, err := store.Query(ContextQuery{Type: ContextTypeFile})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
}

func TestContextStore_Query_TimeRange(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	now := time.Now()

	store.Store(&ContextEntry{Type: ContextTypeFile, Timestamp: now.Add(-2 * time.Hour), Content: "old"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Timestamp: now.Add(-30 * time.Minute), Content: "recent"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Timestamp: now.Add(-10 * time.Minute), Content: "very recent"})

	history, err := store.Query(ContextQuery{
		Since: now.Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
}

func TestContextStore_Query_Pagination(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	// Store 10 entries
	for i := 0; i < 10; i++ {
		store.Store(&ContextEntry{Type: ContextTypeFile, Content: "content"})
	}

	// First page
	history, err := store.Query(ContextQuery{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(history.Entries))
	}
	if history.TotalCount != 10 {
		t.Errorf("Expected total count 10, got %d", history.TotalCount)
	}
	if !history.HasMore {
		t.Error("Expected HasMore to be true")
	}

	// Last page
	history, err = store.Query(ContextQuery{Limit: 3, Offset: 9})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(history.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(history.Entries))
	}
	if history.HasMore {
		t.Error("Expected HasMore to be false")
	}
}

func TestContextStore_Delete(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	entry := &ContextEntry{
		Type:    ContextTypeFile,
		Source:  "test",
		Content: "test content",
		Project: "test-project",
		Topics:  []string{"golang"},
	}
	store.Store(entry)

	err := store.Delete(entry.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(entry.ID)
	if err != ErrEntryNotFound {
		t.Errorf("Expected ErrEntryNotFound after delete, got %v", err)
	}

	// Verify indexes are cleaned up
	history, _ := store.Query(ContextQuery{Project: "test-project"})
	if len(history.Entries) != 0 {
		t.Error("Expected no entries in project index after delete")
	}
}

func TestContextStore_ExpireOldEntries(t *testing.T) {
	store := NewContextStore(100 * time.Millisecond)
	defer store.Stop()

	entry := &ContextEntry{
		Type:    ContextTypeFile,
		Content: "test content",
	}
	store.Store(entry)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	expired := store.ExpireOldEntries()
	if expired != 1 {
		t.Errorf("Expected 1 expired entry, got %d", expired)
	}

	if store.Count() != 0 {
		t.Errorf("Expected 0 entries after expiration, got %d", store.Count())
	}
}

func TestContextStore_GetStats(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "p1", Topics: []string{"t1"}})
	store.Store(&ContextEntry{Type: ContextTypeActivity, Project: "p2", Topics: []string{"t1", "t2"}})

	stats := store.GetStats()

	if stats["total_entries"].(int) != 2 {
		t.Errorf("Expected 2 total entries, got %v", stats["total_entries"])
	}
	if stats["projects_count"].(int) != 2 {
		t.Errorf("Expected 2 projects, got %v", stats["projects_count"])
	}
	if stats["topics_count"].(int) != 2 {
		t.Errorf("Expected 2 topics, got %v", stats["topics_count"])
	}
}
