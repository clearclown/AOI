# TDD Evidence: ACL Manager

## Feature: Access Control List Management
- Component: `backend/internal/acl`
- Date: 2026-01-28

## RED Phase
Test written first: `TestAclManager_CheckPermission_Allow`
```go
func TestAclManager_CheckPermission_Allow(t *testing.T) {
    mgr := NewACLManager()
    mgr.AddRule(ACLRule{AgentID: "agent-1", Resource: "/repo/main", Permission: PermissionRead})
    result := mgr.CheckPermission("agent-1", "/repo/main", "read")
    if !result.Allowed {
        t.Fatal("expected permission to be allowed")
    }
}
```
Result: FAIL - `NewACLManager` undefined, `ACLRule` struct not defined.

## GREEN Phase
Minimal implementation:
```go
type ACLManager struct {
    mu    sync.RWMutex
    rules []ACLRule
}

type ACLRule struct {
    AgentID    string
    Resource   string
    Permission Permission
}

func (m *ACLManager) CheckPermission(agentID, resource, action string) ACLResult {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for _, rule := range m.rules {
        if rule.AgentID == agentID && rule.Resource == resource {
            return ACLResult{Allowed: rule.Permission.Allows(action)}
        }
    }
    return ACLResult{Allowed: false}
}
```
Result: PASS

## REFACTOR Phase
- Extracted Permission levels (None, Read, Write, Admin)
- Added `Allows()` method with permission hierarchy
- Added reason messages to ACLResult
- Added table-driven tests for permission matrix (9 scenarios)
- Added concurrent access tests

## Final Test Count: 28 tests (including 9 subtests)
## Coverage: 100.0%
