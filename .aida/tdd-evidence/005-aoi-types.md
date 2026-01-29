# TDD Evidence: AOI Protocol Types

## Feature: Core Data Types and Serialization
- Component: `backend/pkg/aoi`
- Date: 2026-01-28

## RED Phase
Test written first: `TestAgentIdentity_JSON`
```go
func TestAgentIdentity_JSON(t *testing.T) {
    agent := AgentIdentity{ID: "agent-1", Role: RoleEngineer, Status: "online"}
    data, err := json.Marshal(agent)
    if err != nil {
        t.Fatalf("marshal error: %v", err)
    }
    var decoded AgentIdentity
    json.Unmarshal(data, &decoded)
    if decoded.ID != "agent-1" {
        t.Fatalf("expected agent-1, got %s", decoded.ID)
    }
}
```
Result: FAIL - `AgentIdentity` struct not defined, role constants missing.

## GREEN Phase
```go
type AgentRole string

const (
    RoleEngineer AgentRole = "engineer"
    RolePM       AgentRole = "pm"
)

type AgentIdentity struct {
    ID     string    `json:"id"`
    Role   AgentRole `json:"role"`
    Status string    `json:"status"`
}
```
Result: PASS

## REFACTOR Phase
- Added Query, QueryResult, Task, TaskResult types
- Added HealthResponse type
- Added Metadata map support
- Added ContextScope for scoped queries
- Added JSON tags for all fields
- Validated all role constants

## Final Test Count: 16 tests
## Coverage: 100% (type definitions only)
