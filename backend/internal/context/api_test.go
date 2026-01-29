package context

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func setupTestAPI() (*ContextAPI, *ContextMonitor, *ContextStore, func()) {
	store := NewContextStore(1 * time.Hour)
	monitor := NewContextMonitor(store)
	api := NewContextAPI(monitor, store)

	cleanup := func() {
		store.Stop()
	}

	return api, monitor, store, cleanup
}

func TestContextAPI_HandleContext(t *testing.T) {
	api, monitor, _, cleanup := setupTestAPI()
	defer cleanup()

	monitor.SetActiveProject("test-project")
	monitor.SetActiveFiles([]string{"/path/to/file.go"})

	req := httptest.NewRequest("GET", "/api/v1/context", nil)
	w := httptest.NewRecorder()

	api.handleContext(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var summary ContextSummary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if summary.ActiveProject != "test-project" {
		t.Errorf("Expected active project 'test-project', got '%s'", summary.ActiveProject)
	}
}

func TestContextAPI_HandleContext_MethodNotAllowed(t *testing.T) {
	api, _, _, cleanup := setupTestAPI()
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v1/context", nil)
	w := httptest.NewRecorder()

	api.handleContext(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestContextAPI_HandleContextHistory(t *testing.T) {
	api, _, store, cleanup := setupTestAPI()
	defer cleanup()

	// Add some entries
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-a", Content: "content 1"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-a", Content: "content 2"})
	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "project-b", Content: "content 3"})

	req := httptest.NewRequest("GET", "/api/v1/context/history?project=project-a", nil)
	w := httptest.NewRecorder()

	api.handleContextHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var history ContextHistory
	if err := json.NewDecoder(w.Body).Decode(&history); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
}

func TestContextAPI_HandleContextHistory_Pagination(t *testing.T) {
	api, _, store, cleanup := setupTestAPI()
	defer cleanup()

	// Add 5 entries
	for i := 0; i < 5; i++ {
		store.Store(&ContextEntry{Type: ContextTypeFile, Content: "content"})
	}

	req := httptest.NewRequest("GET", "/api/v1/context/history?limit=2&offset=1", nil)
	w := httptest.NewRecorder()

	api.handleContextHistory(w, req)

	var history ContextHistory
	json.NewDecoder(w.Body).Decode(&history)

	if len(history.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(history.Entries))
	}
	if history.TotalCount != 5 {
		t.Errorf("Expected total count 5, got %d", history.TotalCount)
	}
	if history.Offset != 1 {
		t.Errorf("Expected offset 1, got %d", history.Offset)
	}
}

func TestContextAPI_HandleContextWatch_POST(t *testing.T) {
	api, _, _, cleanup := setupTestAPI()
	defer cleanup()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	body, _ := json.Marshal(WatchRequest{
		Path:      tmpDir,
		Recursive: true,
	})

	req := httptest.NewRequest("POST", "/api/v1/context/watch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	api.handleContextWatch(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp WatchResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Watching {
		t.Error("Expected Watching to be true")
	}
}

func TestContextAPI_HandleContextWatch_GET(t *testing.T) {
	api, monitor, _, cleanup := setupTestAPI()
	defer cleanup()

	// Create and watch a temp directory
	tmpDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	monitor.AddWatch(WatchRequest{Path: tmpDir})

	req := httptest.NewRequest("GET", "/api/v1/context/watch", nil)
	w := httptest.NewRecorder()

	api.handleContextWatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if count := int(resp["count"].(float64)); count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestContextAPI_HandleContextWatch_DELETE(t *testing.T) {
	api, monitor, _, cleanup := setupTestAPI()
	defer cleanup()

	// Create and watch a temp directory
	tmpDir, err := os.MkdirTemp("", "api-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	monitor.AddWatch(WatchRequest{Path: tmpDir})

	req := httptest.NewRequest("DELETE", "/api/v1/context/watch?path="+tmpDir, nil)
	w := httptest.NewRecorder()

	api.handleContextWatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	dirs := monitor.GetWatchedDirs()
	if len(dirs) != 0 {
		t.Errorf("Expected 0 watched dirs, got %d", len(dirs))
	}
}

func TestContextAPI_HandleContextStats(t *testing.T) {
	api, _, store, cleanup := setupTestAPI()
	defer cleanup()

	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "p1"})
	store.Store(&ContextEntry{Type: ContextTypeActivity, Project: "p2"})

	req := httptest.NewRequest("GET", "/api/v1/context/stats", nil)
	w := httptest.NewRecorder()

	api.handleContextStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats map[string]interface{}
	json.NewDecoder(w.Body).Decode(&stats)

	if count := int(stats["total_entries"].(float64)); count != 2 {
		t.Errorf("Expected 2 total entries, got %d", count)
	}
}

func TestContextAPI_HandleContextActivity(t *testing.T) {
	api, _, store, cleanup := setupTestAPI()
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "test",
		"description": "Test activity",
		"metadata":    map[string]interface{}{"key": "value"},
	})

	req := httptest.NewRequest("POST", "/api/v1/context/activity", bytes.NewReader(body))
	w := httptest.NewRecorder()

	api.handleContextActivity(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	history, _ := store.Query(ContextQuery{Type: ContextTypeActivity})
	if len(history.Entries) != 1 {
		t.Errorf("Expected 1 activity entry, got %d", len(history.Entries))
	}
}

func TestContextAPI_HandleContextActivity_MissingDescription(t *testing.T) {
	api, _, _, cleanup := setupTestAPI()
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"type": "test",
	})

	req := httptest.NewRequest("POST", "/api/v1/context/activity", bytes.NewReader(body))
	w := httptest.NewRecorder()

	api.handleContextActivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestContextAPI_HandleJSONRPC_Context(t *testing.T) {
	api, monitor, _, cleanup := setupTestAPI()
	defer cleanup()

	monitor.SetActiveProject("rpc-project")

	result, err := api.HandleJSONRPC("aoi.context", nil)
	if err != nil {
		t.Fatalf("HandleJSONRPC failed: %v", err)
	}

	summary, ok := result.(*ContextSummary)
	if !ok {
		t.Fatal("Expected result to be *ContextSummary")
	}

	if summary.ActiveProject != "rpc-project" {
		t.Errorf("Expected active project 'rpc-project', got '%s'", summary.ActiveProject)
	}
}

func TestContextAPI_HandleJSONRPC_History(t *testing.T) {
	api, _, store, cleanup := setupTestAPI()
	defer cleanup()

	store.Store(&ContextEntry{Type: ContextTypeFile, Project: "p1"})

	params, _ := json.Marshal(ContextQuery{Project: "p1"})
	result, err := api.HandleJSONRPC("aoi.context.history", params)
	if err != nil {
		t.Fatalf("HandleJSONRPC failed: %v", err)
	}

	history, ok := result.(*ContextHistory)
	if !ok {
		t.Fatal("Expected result to be *ContextHistory")
	}

	if len(history.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(history.Entries))
	}
}

func TestContextAPI_HandleJSONRPC_UnknownMethod(t *testing.T) {
	api, _, _, cleanup := setupTestAPI()
	defer cleanup()

	_, err := api.HandleJSONRPC("aoi.unknown", nil)
	if err == nil {
		t.Error("Expected error for unknown method")
	}
}

func TestContextAPI_RegisterRoutes(t *testing.T) {
	api, _, _, cleanup := setupTestAPI()
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Test that routes are registered by making requests
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/context")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
