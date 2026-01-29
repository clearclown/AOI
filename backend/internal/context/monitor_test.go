package context

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContextMonitor_NewContextMonitor(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)
	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.store != store {
		t.Error("Expected monitor to have the store reference")
	}
}

func TestContextMonitor_AddWatch(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "context-monitor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	resp, err := monitor.AddWatch(WatchRequest{
		Path:      tmpDir,
		Recursive: true,
		Patterns:  []string{"*.txt", "*.go"},
	})
	if err != nil {
		t.Fatalf("AddWatch failed: %v", err)
	}

	if !resp.Watching {
		t.Error("Expected Watching to be true")
	}

	dirs := monitor.GetWatchedDirs()
	if len(dirs) != 1 {
		t.Errorf("Expected 1 watched dir, got %d", len(dirs))
	}
}

func TestContextMonitor_AddWatch_InvalidPath(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	_, err := monitor.AddWatch(WatchRequest{
		Path: "/nonexistent/path/that/does/not/exist",
	})
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

func TestContextMonitor_AddWatch_NotDirectory(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "context-monitor-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = monitor.AddWatch(WatchRequest{
		Path: tmpFile.Name(),
	})
	if err == nil {
		t.Error("Expected error for non-directory path")
	}
}

func TestContextMonitor_RemoveWatch(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "context-monitor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	monitor.AddWatch(WatchRequest{Path: tmpDir})

	err = monitor.RemoveWatch(tmpDir)
	if err != nil {
		t.Fatalf("RemoveWatch failed: %v", err)
	}

	dirs := monitor.GetWatchedDirs()
	if len(dirs) != 0 {
		t.Errorf("Expected 0 watched dirs, got %d", len(dirs))
	}
}

func TestContextMonitor_RemoveWatch_NotWatched(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	err := monitor.RemoveWatch("/some/path")
	if err == nil {
		t.Error("Expected error for unwatched path")
	}
}

func TestContextMonitor_SetActiveProject(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)
	monitor.SetActiveProject("my-project")

	summary := monitor.GetSummary()
	if summary.ActiveProject != "my-project" {
		t.Errorf("Expected active project 'my-project', got '%s'", summary.ActiveProject)
	}
}

func TestContextMonitor_SetActiveFiles(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)
	files := []string{"/path/to/file1.go", "/path/to/file2.go"}
	monitor.SetActiveFiles(files)

	summary := monitor.GetSummary()
	if len(summary.ActiveFiles) != 2 {
		t.Errorf("Expected 2 active files, got %d", len(summary.ActiveFiles))
	}
}

func TestContextMonitor_GetSummary(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	// Add some context entries
	store.Store(&ContextEntry{
		Type:    ContextTypeActivity,
		Summary: "Test activity",
		Topics:  []string{"testing"},
	})

	summary := monitor.GetSummary()
	if summary.TotalEntries != 1 {
		t.Errorf("Expected 1 total entry, got %d", summary.TotalEntries)
	}
}

func TestContextMonitor_RecordActivity(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)
	monitor.SetActiveProject("test-project")

	err := monitor.RecordActivity("test", "Test activity description", map[string]any{
		"topics": []string{"testing", "manual"},
	})
	if err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}

	history, _ := store.Query(ContextQuery{Type: ContextTypeActivity})
	if len(history.Entries) != 1 {
		t.Errorf("Expected 1 activity entry, got %d", len(history.Entries))
	}
}

func TestContextMonitor_inferProjectFromPath(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	// Create a temp directory with go.mod
	tmpDir, err := os.MkdirTemp("", "test-project")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	os.WriteFile(goModPath, []byte("module test"), 0644)

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "internal")
	os.MkdirAll(subDir, 0755)
	testFile := filepath.Join(subDir, "test.go")
	os.WriteFile(testFile, []byte("package internal"), 0644)

	project := monitor.inferProjectFromPath(testFile)
	if project != filepath.Base(tmpDir) {
		t.Errorf("Expected project '%s', got '%s'", filepath.Base(tmpDir), project)
	}
}

func TestContextMonitor_inferTopicsFromPath(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	tests := []struct {
		path     string
		expected []string
	}{
		{"/path/to/file.go", []string{"golang", "backend"}},
		{"/path/to/file.ts", []string{"javascript", "frontend"}},
		{"/path/to/file.py", []string{"python"}},
		{"/path/to/README.md", []string{"documentation"}},
		{"/path/to/test_file.go", []string{"golang", "backend", "testing"}},
		{"/path/to/internal/service.go", []string{"golang", "backend", "library"}},
	}

	for _, tt := range tests {
		topics := monitor.inferTopicsFromPath(tt.path)
		for _, expected := range tt.expected {
			found := false
			for _, topic := range topics {
				if topic == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Path %s: expected topic '%s' not found in %v", tt.path, expected, topics)
			}
		}
	}
}

func TestContextMonitor_matchesPatterns(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)

	tests := []struct {
		name     string
		patterns []string
		expected bool
	}{
		{"file.go", []string{"*.go"}, true},
		{"file.txt", []string{"*.go"}, false},
		{"file.go", []string{"*"}, true},
		{"file.go", []string{"*.txt", "*.go"}, true},
		{"test.txt", []string{"test.*"}, true},
	}

	for _, tt := range tests {
		result := monitor.matchesPatterns(tt.name, tt.patterns)
		if result != tt.expected {
			t.Errorf("matchesPatterns(%s, %v) = %v, expected %v", tt.name, tt.patterns, result, tt.expected)
		}
	}
}

func TestContextMonitor_StartStop(t *testing.T) {
	store := NewContextStore(1 * time.Hour)
	defer store.Stop()

	monitor := NewContextMonitor(store)
	monitor.SetPollInterval(100 * time.Millisecond)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Should be safe to call Stop again
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Second Stop failed: %v", err)
	}
}
