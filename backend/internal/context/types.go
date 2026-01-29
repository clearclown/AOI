package context

import (
	"time"
)

// ContextEntryType represents the type of context entry
type ContextEntryType string

const (
	// ContextTypeFile represents a file change context
	ContextTypeFile ContextEntryType = "file"
	// ContextTypeProject represents a project-level context
	ContextTypeProject ContextEntryType = "project"
	// ContextTypeActivity represents an activity context
	ContextTypeActivity ContextEntryType = "activity"
	// ContextTypeTopic represents a topic-based context
	ContextTypeTopic ContextEntryType = "topic"
)

// ContextEntry represents a single context entry with metadata
type ContextEntry struct {
	ID        string           `json:"id"`
	Type      ContextEntryType `json:"type"`
	Source    string           `json:"source"`
	Content   string           `json:"content"`
	Summary   string           `json:"summary"`
	Project   string           `json:"project,omitempty"`
	File      string           `json:"file,omitempty"`
	Topics    []string         `json:"topics,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
	ExpiresAt time.Time        `json:"expires_at,omitempty"`
	Metadata  map[string]any   `json:"metadata,omitempty"`
}

// ContextSummary represents a summarized view of current context
type ContextSummary struct {
	ActiveProject   string            `json:"active_project,omitempty"`
	ActiveFiles     []string          `json:"active_files,omitempty"`
	RecentActivity  []ActivitySummary `json:"recent_activity,omitempty"`
	Topics          []string          `json:"topics,omitempty"`
	TotalEntries    int               `json:"total_entries"`
	LastUpdated     time.Time         `json:"last_updated"`
	WatchedDirs     []string          `json:"watched_dirs,omitempty"`
}

// ActivitySummary represents a summary of recent activity
type ActivitySummary struct {
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	File        string    `json:"file,omitempty"`
	Project     string    `json:"project,omitempty"`
}

// ContextQuery represents parameters for querying context
type ContextQuery struct {
	Project   string           `json:"project,omitempty"`
	File      string           `json:"file,omitempty"`
	Topic     string           `json:"topic,omitempty"`
	Type      ContextEntryType `json:"type,omitempty"`
	Since     time.Time        `json:"since,omitempty"`
	Until     time.Time        `json:"until,omitempty"`
	Limit     int              `json:"limit,omitempty"`
	Offset    int              `json:"offset,omitempty"`
}

// ContextHistory represents a list of context entries with pagination
type ContextHistory struct {
	Entries    []ContextEntry `json:"entries"`
	TotalCount int            `json:"total_count"`
	Offset     int            `json:"offset"`
	Limit      int            `json:"limit"`
	HasMore    bool           `json:"has_more"`
}

// WatchRequest represents a request to add a directory to watch
type WatchRequest struct {
	Path         string   `json:"path"`
	Recursive    bool     `json:"recursive"`
	Patterns     []string `json:"patterns,omitempty"`     // File patterns to watch (e.g., "*.go", "*.md")
	IgnoreHidden bool     `json:"ignore_hidden"`
}

// WatchResponse represents the response after adding a watch
type WatchResponse struct {
	Path      string    `json:"path"`
	Watching  bool      `json:"watching"`
	Message   string    `json:"message,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}

// FileChangeEvent represents a detected file change
type FileChangeEvent struct {
	Path      string    `json:"path"`
	Operation string    `json:"operation"` // "create", "modify", "delete", "rename"
	OldPath   string    `json:"old_path,omitempty"` // For rename operations
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size,omitempty"`
}
