package context

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrEntryNotFound is returned when a context entry is not found
	ErrEntryNotFound = errors.New("context entry not found")
	// ErrEntryExpired is returned when a context entry has expired
	ErrEntryExpired = errors.New("context entry expired")
)

// ContextStore stores and indexes context entries
type ContextStore struct {
	mu            sync.RWMutex
	entries       map[string]*ContextEntry // ID -> Entry
	projectIndex  map[string][]string      // Project -> Entry IDs
	fileIndex     map[string][]string      // File -> Entry IDs
	topicIndex    map[string][]string      // Topic -> Entry IDs
	typeIndex     map[ContextEntryType][]string // Type -> Entry IDs
	
	defaultTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewContextStore creates a new context store with the given default TTL
func NewContextStore(defaultTTL time.Duration) *ContextStore {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour // Default to 24 hours
	}

	cs := &ContextStore{
		entries:       make(map[string]*ContextEntry),
		projectIndex:  make(map[string][]string),
		fileIndex:     make(map[string][]string),
		topicIndex:    make(map[string][]string),
		typeIndex:     make(map[ContextEntryType][]string),
		defaultTTL:    defaultTTL,
		stopCleanup:   make(chan struct{}),
	}

	// Start cleanup goroutine
	cs.cleanupTicker = time.NewTicker(5 * time.Minute)
	go cs.cleanupLoop()

	return cs
}

// cleanupLoop periodically removes expired entries
func (cs *ContextStore) cleanupLoop() {
	for {
		select {
		case <-cs.cleanupTicker.C:
			cs.ExpireOldEntries()
		case <-cs.stopCleanup:
			cs.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (cs *ContextStore) Stop() {
	close(cs.stopCleanup)
}

// Store adds a new context entry to the store
func (cs *ContextStore) Store(entry *ContextEntry) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Set expiration if not provided
	if entry.ExpiresAt.IsZero() {
		entry.ExpiresAt = time.Now().Add(cs.defaultTTL)
	}

	// Store the entry
	cs.entries[entry.ID] = entry

	// Update indexes
	if entry.Project != "" {
		cs.projectIndex[entry.Project] = append(cs.projectIndex[entry.Project], entry.ID)
	}
	if entry.File != "" {
		cs.fileIndex[entry.File] = append(cs.fileIndex[entry.File], entry.ID)
	}
	for _, topic := range entry.Topics {
		cs.topicIndex[topic] = append(cs.topicIndex[topic], entry.ID)
	}
	cs.typeIndex[entry.Type] = append(cs.typeIndex[entry.Type], entry.ID)

	return nil
}

// Get retrieves a context entry by ID
func (cs *ContextStore) Get(id string) (*ContextEntry, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	entry, ok := cs.entries[id]
	if !ok {
		return nil, ErrEntryNotFound
	}

	// Check if expired
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return nil, ErrEntryExpired
	}

	return entry, nil
}

// Query retrieves context entries matching the given query parameters
func (cs *ContextStore) Query(q ContextQuery) (*ContextHistory, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Set defaults
	if q.Limit <= 0 {
		q.Limit = 100
	}

	// Collect candidate entry IDs
	var candidateIDs []string
	
	// Start with all entries if no specific index filter
	if q.Project == "" && q.File == "" && q.Topic == "" && q.Type == "" {
		for id := range cs.entries {
			candidateIDs = append(candidateIDs, id)
		}
	} else {
		// Use indexes to narrow down
		idSets := make([]map[string]bool, 0)

		if q.Project != "" {
			idSet := make(map[string]bool)
			for _, id := range cs.projectIndex[q.Project] {
				idSet[id] = true
			}
			idSets = append(idSets, idSet)
		}
		if q.File != "" {
			idSet := make(map[string]bool)
			for _, id := range cs.fileIndex[q.File] {
				idSet[id] = true
			}
			idSets = append(idSets, idSet)
		}
		if q.Topic != "" {
			idSet := make(map[string]bool)
			for _, id := range cs.topicIndex[q.Topic] {
				idSet[id] = true
			}
			idSets = append(idSets, idSet)
		}
		if q.Type != "" {
			idSet := make(map[string]bool)
			for _, id := range cs.typeIndex[q.Type] {
				idSet[id] = true
			}
			idSets = append(idSets, idSet)
		}

		// Intersect all ID sets
		if len(idSets) > 0 {
			candidateIDs = intersectIDSets(idSets)
		}
	}

	// Filter by time range and collect entries
	var filteredEntries []ContextEntry
	now := time.Now()

	for _, id := range candidateIDs {
		entry := cs.entries[id]
		if entry == nil {
			continue
		}

		// Skip expired entries
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			continue
		}

		// Check time range
		if !q.Since.IsZero() && entry.Timestamp.Before(q.Since) {
			continue
		}
		if !q.Until.IsZero() && entry.Timestamp.After(q.Until) {
			continue
		}

		filteredEntries = append(filteredEntries, *entry)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(filteredEntries, func(i, j int) bool {
		return filteredEntries[i].Timestamp.After(filteredEntries[j].Timestamp)
	})

	// Apply pagination
	totalCount := len(filteredEntries)
	start := q.Offset
	if start > totalCount {
		start = totalCount
	}
	end := start + q.Limit
	if end > totalCount {
		end = totalCount
	}

	result := filteredEntries[start:end]

	return &ContextHistory{
		Entries:    result,
		TotalCount: totalCount,
		Offset:     q.Offset,
		Limit:      q.Limit,
		HasMore:    end < totalCount,
	}, nil
}

// GetByProject retrieves all context entries for a project
func (cs *ContextStore) GetByProject(project string) ([]ContextEntry, error) {
	history, err := cs.Query(ContextQuery{Project: project, Limit: 1000})
	if err != nil {
		return nil, err
	}
	return history.Entries, nil
}

// GetByFile retrieves all context entries for a file
func (cs *ContextStore) GetByFile(file string) ([]ContextEntry, error) {
	history, err := cs.Query(ContextQuery{File: file, Limit: 1000})
	if err != nil {
		return nil, err
	}
	return history.Entries, nil
}

// GetByTopic retrieves all context entries for a topic
func (cs *ContextStore) GetByTopic(topic string) ([]ContextEntry, error) {
	history, err := cs.Query(ContextQuery{Topic: topic, Limit: 1000})
	if err != nil {
		return nil, err
	}
	return history.Entries, nil
}

// Delete removes a context entry by ID
func (cs *ContextStore) Delete(id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.entries[id]
	if !ok {
		return ErrEntryNotFound
	}

	// Remove from indexes
	cs.removeFromIndex(cs.projectIndex, entry.Project, id)
	cs.removeFromIndex(cs.fileIndex, entry.File, id)
	for _, topic := range entry.Topics {
		cs.removeFromIndex(cs.topicIndex, topic, id)
	}
	cs.removeFromTypeIndex(entry.Type, id)

	// Delete entry
	delete(cs.entries, id)

	return nil
}

// ExpireOldEntries removes all expired entries from the store
func (cs *ContextStore) ExpireOldEntries() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, entry := range cs.entries {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			// Remove from indexes
			cs.removeFromIndex(cs.projectIndex, entry.Project, id)
			cs.removeFromIndex(cs.fileIndex, entry.File, id)
			for _, topic := range entry.Topics {
				cs.removeFromIndex(cs.topicIndex, topic, id)
			}
			cs.removeFromTypeIndex(entry.Type, id)

			// Delete entry
			delete(cs.entries, id)
			expiredCount++
		}
	}

	return expiredCount
}

// Count returns the total number of entries in the store
func (cs *ContextStore) Count() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.entries)
}

// GetStats returns statistics about the store
func (cs *ContextStore) GetStats() map[string]any {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	stats := map[string]any{
		"total_entries":   len(cs.entries),
		"projects_count":  len(cs.projectIndex),
		"files_count":     len(cs.fileIndex),
		"topics_count":    len(cs.topicIndex),
	}

	// Count by type
	typeStats := make(map[string]int)
	for entryType, ids := range cs.typeIndex {
		typeStats[string(entryType)] = len(ids)
	}
	stats["entries_by_type"] = typeStats

	return stats
}

// Helper function to remove an ID from a string-keyed index
func (cs *ContextStore) removeFromIndex(index map[string][]string, key, id string) {
	if key == "" {
		return
	}
	ids := index[key]
	for i, existingID := range ids {
		if existingID == id {
			index[key] = append(ids[:i], ids[i+1:]...)
			if len(index[key]) == 0 {
				delete(index, key)
			}
			return
		}
	}
}

// Helper function to remove an ID from the type index
func (cs *ContextStore) removeFromTypeIndex(entryType ContextEntryType, id string) {
	ids := cs.typeIndex[entryType]
	for i, existingID := range ids {
		if existingID == id {
			cs.typeIndex[entryType] = append(ids[:i], ids[i+1:]...)
			if len(cs.typeIndex[entryType]) == 0 {
				delete(cs.typeIndex, entryType)
			}
			return
		}
	}
}

// Helper function to intersect multiple ID sets
func intersectIDSets(sets []map[string]bool) []string {
	if len(sets) == 0 {
		return nil
	}

	// Start with the smallest set for efficiency
	smallest := 0
	for i, s := range sets {
		if len(s) < len(sets[smallest]) {
			smallest = i
		}
	}

	var result []string
	for id := range sets[smallest] {
		inAll := true
		for i, s := range sets {
			if i != smallest && !s[id] {
				inAll = false
				break
			}
		}
		if inAll {
			result = append(result, id)
		}
	}

	return result
}
