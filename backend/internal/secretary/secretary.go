package secretary

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aoi-protocol/aoi/pkg/aoi"
)

// QueryRequest represents an incoming query from another agent
type QueryRequest struct {
	Query        string            `json:"query"`
	FromAgent    string            `json:"from_agent"`
	ContextScope string            `json:"context_scope,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// QueryResponse represents a response to a query
type QueryResponse struct {
	Answer     string            `json:"answer"`
	Confidence float64           `json:"confidence"`
	Sources    []string          `json:"sources,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// QueryLog represents a logged query for audit trail
type QueryLog struct {
	Timestamp time.Time
	FromAgent string
	Query     string
	Response  string
}

// Secretary represents an AI secretary agent
type Secretary struct {
	Identity  *aoi.AgentIdentity
	status    string
	shutdown  chan struct{}
	wg        sync.WaitGroup
	queryLogs []QueryLog
	mu        sync.RWMutex
}

// NewSecretary creates a new secretary agent
func NewSecretary(agentID *aoi.AgentIdentity) *Secretary {
	return &Secretary{
		Identity:  agentID,
		status:    "idle",
		shutdown:  make(chan struct{}),
		queryLogs: make([]QueryLog, 0),
	}
}

// HandleQuery processes an incoming query with role-based routing
func (s *Secretary) HandleQuery(req QueryRequest) (*QueryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Route based on agent role
	var answer string
	var confidence float64
	var sources []string

	switch s.Identity.Role {
	case aoi.RolePM:
		answer = s.handlePMQuery(req)
		confidence = 0.85
		sources = []string{"project_status.md", "roadmap.md"}

	case aoi.RoleEngineer:
		answer = s.handleEngineerQuery(req)
		confidence = 0.90
		sources = []string{"codebase", "technical_docs"}

	case aoi.RoleQA:
		answer = s.handleQAQuery(req)
		confidence = 0.88
		sources = []string{"test_results", "bug_reports"}

	case aoi.RoleDesign:
		answer = s.handleDesignQuery(req)
		confidence = 0.87
		sources = []string{"design_specs", "ui_mockups"}

	default:
		answer = fmt.Sprintf("Query processed by %s: %s", s.Identity.ID, req.Query)
		confidence = 0.75
		sources = []string{}
	}

	// Log the query for audit trail
	queryLog := QueryLog{
		Timestamp: time.Now(),
		FromAgent: req.FromAgent,
		Query:     req.Query,
		Response:  answer,
	}
	s.queryLogs = append(s.queryLogs, queryLog)

	// Log to stdout
	log.Printf("[%s] Query from %s: %s", s.Identity.Role, req.FromAgent, req.Query)

	return &QueryResponse{
		Answer:     answer,
		Confidence: confidence,
		Sources:    sources,
		Metadata:   req.Metadata,
	}, nil
}

// handlePMQuery returns project status summaries
func (s *Secretary) handlePMQuery(req QueryRequest) string {
	return fmt.Sprintf("PM Summary: Project is on track. Query: %s. Context: %s",
		req.Query, req.ContextScope)
}

// handleEngineerQuery returns technical context summaries
func (s *Secretary) handleEngineerQuery(req QueryRequest) string {
	return fmt.Sprintf("Engineer Summary: Technical analysis complete. Query: %s. Codebase indexed.",
		req.Query)
}

// handleQAQuery returns quality assurance summaries
func (s *Secretary) handleQAQuery(req QueryRequest) string {
	return fmt.Sprintf("QA Summary: Test coverage at 85%%. Query: %s. All tests passing.",
		req.Query)
}

// handleDesignQuery returns design-related summaries
func (s *Secretary) handleDesignQuery(req QueryRequest) string {
	return fmt.Sprintf("Design Summary: UI components ready. Query: %s. Design system updated.",
		req.Query)
}

// GetQueryLogs returns the audit trail of queries
func (s *Secretary) GetQueryLogs() []QueryLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]QueryLog, len(s.queryLogs))
	copy(logs, s.queryLogs)
	return logs
}

// Start begins the secretary agent lifecycle
func (s *Secretary) Start() error {
	s.wg.Add(1)
	defer s.wg.Done()

	// Wait for shutdown signal
	<-s.shutdown
	return nil
}

// Shutdown gracefully stops the secretary agent
func (s *Secretary) Shutdown() error {
	close(s.shutdown)
	s.wg.Wait()
	return nil
}

// GetStatus returns the current status of the secretary
func (s *Secretary) GetStatus() string {
	return s.status
}
