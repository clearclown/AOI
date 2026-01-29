package context

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ContextMonitor watches directories and tracks file changes
type ContextMonitor struct {
	mu            sync.RWMutex
	store         *ContextStore
	watchDirs     map[string]*WatchConfig
	fileHashes    map[string]string // path -> hash for change detection
	activeProject string
	activeFiles   []string
	
	pollInterval  time.Duration
	pollTicker    *time.Ticker
	stopChan      chan struct{}
	eventChan     chan FileChangeEvent
	running       bool
}

// WatchConfig holds configuration for a watched directory
type WatchConfig struct {
	Path         string
	Recursive    bool
	Patterns     []string
	IgnoreHidden bool
	AddedAt      time.Time
}

// NewContextMonitor creates a new context monitor
func NewContextMonitor(store *ContextStore) *ContextMonitor {
	return &ContextMonitor{
		store:        store,
		watchDirs:    make(map[string]*WatchConfig),
		fileHashes:   make(map[string]string),
		pollInterval: 5 * time.Second,
		eventChan:    make(chan FileChangeEvent, 100),
		stopChan:     make(chan struct{}),
	}
}

// SetPollInterval sets the polling interval for file change detection
func (cm *ContextMonitor) SetPollInterval(interval time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.pollInterval = interval
}

// Start begins monitoring watched directories
func (cm *ContextMonitor) Start() error {
	cm.mu.Lock()
	if cm.running {
		cm.mu.Unlock()
		return nil
	}
	cm.running = true
	cm.pollTicker = time.NewTicker(cm.pollInterval)
	cm.mu.Unlock()

	log.Printf("[ContextMonitor] Started with poll interval: %v", cm.pollInterval)

	go cm.pollLoop()
	go cm.processEvents()

	return nil
}

// Stop stops the context monitor
func (cm *ContextMonitor) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return nil
	}

	cm.running = false
	close(cm.stopChan)
	if cm.pollTicker != nil {
		cm.pollTicker.Stop()
	}

	log.Printf("[ContextMonitor] Stopped")
	return nil
}

// AddWatch adds a directory to watch
func (cm *ContextMonitor) AddWatch(req WatchRequest) (*WatchResponse, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate path
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Default patterns if not provided
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"*"} // Watch all files
	}

	config := &WatchConfig{
		Path:         absPath,
		Recursive:    req.Recursive,
		Patterns:     patterns,
		IgnoreHidden: req.IgnoreHidden,
		AddedAt:      time.Now(),
	}

	cm.watchDirs[absPath] = config

	// Initial scan
	go cm.scanDirectory(config)

	log.Printf("[ContextMonitor] Added watch: %s (recursive=%v, patterns=%v)", absPath, req.Recursive, patterns)

	return &WatchResponse{
		Path:     absPath,
		Watching: true,
		Message:  fmt.Sprintf("Now watching %s", absPath),
		AddedAt:  config.AddedAt,
	}, nil
}

// RemoveWatch removes a directory from watching
func (cm *ContextMonitor) RemoveWatch(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, ok := cm.watchDirs[absPath]; !ok {
		return fmt.Errorf("path not being watched: %s", absPath)
	}

	delete(cm.watchDirs, absPath)
	log.Printf("[ContextMonitor] Removed watch: %s", absPath)

	return nil
}

// GetWatchedDirs returns the list of watched directories
func (cm *ContextMonitor) GetWatchedDirs() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	dirs := make([]string, 0, len(cm.watchDirs))
	for dir := range cm.watchDirs {
		dirs = append(dirs, dir)
	}
	return dirs
}

// SetActiveProject sets the currently active project
func (cm *ContextMonitor) SetActiveProject(project string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeProject = project
}

// SetActiveFiles sets the currently active files
func (cm *ContextMonitor) SetActiveFiles(files []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeFiles = make([]string, len(files))
	copy(cm.activeFiles, files)
}

// GetSummary returns a summary of current context
func (cm *ContextMonitor) GetSummary() *ContextSummary {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Get recent activity from store
	history, _ := cm.store.Query(ContextQuery{
		Limit: 10,
		Since: time.Now().Add(-24 * time.Hour),
	})

	var recentActivity []ActivitySummary
	if history != nil {
		for _, entry := range history.Entries {
			recentActivity = append(recentActivity, ActivitySummary{
				Description: entry.Summary,
				Type:        string(entry.Type),
				Timestamp:   entry.Timestamp,
				File:        entry.File,
				Project:     entry.Project,
			})
		}
	}

	// Collect unique topics
	topicSet := make(map[string]bool)
	if history != nil {
		for _, entry := range history.Entries {
			for _, topic := range entry.Topics {
				topicSet[topic] = true
			}
		}
	}
	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return &ContextSummary{
		ActiveProject:  cm.activeProject,
		ActiveFiles:    cm.activeFiles,
		RecentActivity: recentActivity,
		Topics:         topics,
		TotalEntries:   cm.store.Count(),
		LastUpdated:    time.Now(),
		WatchedDirs:    cm.GetWatchedDirs(),
	}
}

// pollLoop periodically scans watched directories for changes
func (cm *ContextMonitor) pollLoop() {
	for {
		select {
		case <-cm.pollTicker.C:
			cm.scanAllDirectories()
		case <-cm.stopChan:
			return
		}
	}
}

// scanAllDirectories scans all watched directories
func (cm *ContextMonitor) scanAllDirectories() {
	cm.mu.RLock()
	configs := make([]*WatchConfig, 0, len(cm.watchDirs))
	for _, config := range cm.watchDirs {
		configs = append(configs, config)
	}
	cm.mu.RUnlock()

	for _, config := range configs {
		cm.scanDirectory(config)
	}
}

// scanDirectory scans a single directory for changes
func (cm *ContextMonitor) scanDirectory(config *WatchConfig) {
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories in the walk if not recursive
		if info.IsDir() {
			if path != config.Path && !config.Recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches patterns
		if !cm.matchesPatterns(info.Name(), config.Patterns) {
			return nil
		}

		// Skip hidden files if configured
		if config.IgnoreHidden && strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Calculate file hash
		hash, err := cm.calculateFileHash(path)
		if err != nil {
			return nil
		}

		cm.mu.Lock()
		oldHash, exists := cm.fileHashes[path]
		cm.fileHashes[path] = hash
		cm.mu.Unlock()

		if !exists {
			// New file
			cm.eventChan <- FileChangeEvent{
				Path:      path,
				Operation: "create",
				Timestamp: info.ModTime(),
				Size:      info.Size(),
			}
		} else if oldHash != hash {
			// Modified file
			cm.eventChan <- FileChangeEvent{
				Path:      path,
				Operation: "modify",
				Timestamp: info.ModTime(),
				Size:      info.Size(),
			}
		}

		return nil
	}

	filepath.Walk(config.Path, walkFn)
}

// processEvents processes file change events
func (cm *ContextMonitor) processEvents() {
	for {
		select {
		case event := <-cm.eventChan:
			cm.handleFileChange(event)
		case <-cm.stopChan:
			return
		}
	}
}

// handleFileChange creates a context entry for a file change
func (cm *ContextMonitor) handleFileChange(event FileChangeEvent) {
	// Determine project from path
	project := cm.inferProjectFromPath(event.Path)

	// Generate summary based on operation
	var summary string
	switch event.Operation {
	case "create":
		summary = fmt.Sprintf("New file created: %s", filepath.Base(event.Path))
	case "modify":
		summary = fmt.Sprintf("File modified: %s", filepath.Base(event.Path))
	case "delete":
		summary = fmt.Sprintf("File deleted: %s", filepath.Base(event.Path))
	case "rename":
		summary = fmt.Sprintf("File renamed from %s to %s", filepath.Base(event.OldPath), filepath.Base(event.Path))
	}

	// Infer topics from file extension and path
	topics := cm.inferTopicsFromPath(event.Path)

	entry := &ContextEntry{
		Type:      ContextTypeFile,
		Source:    "file_monitor",
		Content:   fmt.Sprintf("File change detected: %s (%s)", event.Path, event.Operation),
		Summary:   summary,
		Project:   project,
		File:      event.Path,
		Topics:    topics,
		Timestamp: event.Timestamp,
		Metadata: map[string]any{
			"operation": event.Operation,
			"size":      event.Size,
		},
	}

	if err := cm.store.Store(entry); err != nil {
		log.Printf("[ContextMonitor] Failed to store context entry: %v", err)
	} else {
		log.Printf("[ContextMonitor] Recorded file change: %s (%s)", event.Path, event.Operation)
	}
}

// matchesPatterns checks if a filename matches any of the given patterns
func (cm *ContextMonitor) matchesPatterns(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == "*" {
			return true
		}
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// calculateFileHash calculates a simple hash for change detection
func (cm *ContextMonitor) calculateFileHash(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Use size and mod time for quick change detection
	data := fmt.Sprintf("%d-%d", info.Size(), info.ModTime().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]), nil
}

// inferProjectFromPath attempts to determine the project name from a file path
func (cm *ContextMonitor) inferProjectFromPath(path string) string {
	// Look for common project indicators
	indicators := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", ".git"}
	
	dir := filepath.Dir(path)
	for dir != "/" && dir != "." {
		for _, indicator := range indicators {
			indicatorPath := filepath.Join(dir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				return filepath.Base(dir)
			}
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

// inferTopicsFromPath infers topics based on file path and extension
func (cm *ContextMonitor) inferTopicsFromPath(path string) []string {
	var topics []string

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		topics = append(topics, "golang", "backend")
	case ".js", ".ts", ".jsx", ".tsx":
		topics = append(topics, "javascript", "frontend")
	case ".py":
		topics = append(topics, "python")
	case ".md":
		topics = append(topics, "documentation")
	case ".yaml", ".yml":
		topics = append(topics, "configuration")
	case ".json":
		topics = append(topics, "data", "configuration")
	case ".sql":
		topics = append(topics, "database")
	case ".sh", ".bash":
		topics = append(topics, "scripting")
	case ".dockerfile", "":
		if strings.Contains(strings.ToLower(filepath.Base(path)), "docker") {
			topics = append(topics, "docker", "deployment")
		}
	}

	// Check path components
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, "test") {
		topics = append(topics, "testing")
	}
	if strings.Contains(pathLower, "internal") || strings.Contains(pathLower, "pkg") {
		topics = append(topics, "library")
	}
	if strings.Contains(pathLower, "cmd") || strings.Contains(pathLower, "main") {
		topics = append(topics, "application")
	}

	return topics
}

// RecordActivity manually records an activity in the context
func (cm *ContextMonitor) RecordActivity(activityType string, description string, metadata map[string]any) error {
	entry := &ContextEntry{
		Type:      ContextTypeActivity,
		Source:    "manual",
		Content:   description,
		Summary:   description,
		Project:   cm.activeProject,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	if metadata != nil {
		if topics, ok := metadata["topics"].([]string); ok {
			entry.Topics = topics
		}
	}

	return cm.store.Store(entry)
}
