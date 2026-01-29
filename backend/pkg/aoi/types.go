package aoi

// AgentRole represents the role of an agent
type AgentRole string

const (
	RolePM       AgentRole = "pm"
	RoleEngineer AgentRole = "engineer"
	RoleQA       AgentRole = "qa"
	RoleDesign   AgentRole = "design"
)

// AgentIdentity represents an agent's identity
type AgentIdentity struct {
	ID              string                 `json:"id"`
	Role            AgentRole              `json:"role"`
	Owner           string                 `json:"owner"`
	Capabilities    []string               `json:"capabilities"`
	Endpoint        string                 `json:"endpoint"`
	Status          string                 `json:"status"`
	TailscaleNodeID string                 `json:"tailscale_node_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Query represents a query message
type Query struct {
	ID           string                 `json:"id"`
	From         string                 `json:"from"`
	To           string                 `json:"to"`
	Query        string                 `json:"query"`
	ContextScope []string               `json:"context_scope,omitempty"`
	Priority     string                 `json:"priority"`
	Async        bool                   `json:"async"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// QueryResult represents the response to a query
type QueryResult struct {
	Summary     string                 `json:"summary"`
	Progress    int                    `json:"progress,omitempty"`
	Blockers    []string               `json:"blockers,omitempty"`
	ContextRefs []string               `json:"context_refs,omitempty"`
	Completed   bool                   `json:"completed"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Task represents a task execution request
type Task struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Async      bool                   `json:"async"`
	Timeout    int                    `json:"timeout,omitempty"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID   string                 `json:"task_id"`
	Status   string                 `json:"status"`
	Output   string                 `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}
