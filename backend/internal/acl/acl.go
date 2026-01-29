package acl

import (
	"sync"
)

// PermissionLevel represents the level of access
type PermissionLevel int

const (
	PermissionNone PermissionLevel = iota
	PermissionRead
	PermissionWrite
	PermissionAdmin
)

// AccessRule defines an access control rule
type AccessRule struct {
	AgentID    string
	Resource   string
	Permission PermissionLevel
}

// PermissionCheckResult contains the result of a permission check
type PermissionCheckResult struct {
	Allowed bool
	Reason  string
}

// AclManager manages access control lists
type AclManager struct {
	rules []AccessRule
	mu    sync.RWMutex
}

// NewAclManager creates a new ACL manager
func NewAclManager() *AclManager {
	return &AclManager{
		rules: make([]AccessRule, 0),
	}
}

// AddRule adds a new access rule
func (m *AclManager) AddRule(rule *AccessRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules = append(m.rules, *rule)
}

// CheckPermission checks if an agent has permission for an action
func (m *AclManager) CheckPermission(agentID string, resource string, action string) PermissionCheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find matching rules
	for _, rule := range m.rules {
		if rule.AgentID == agentID && rule.Resource == resource {
			// Check if permission level is sufficient
			allowed := false

			switch action {
			case "read":
				allowed = rule.Permission >= PermissionRead
			case "write":
				allowed = rule.Permission >= PermissionWrite
			case "execute":
				allowed = rule.Permission >= PermissionWrite
			default:
				allowed = false
			}

			if allowed {
				return PermissionCheckResult{Allowed: true, Reason: "permission granted"}
			}
		}
	}

	// No matching rule or insufficient permission
	return PermissionCheckResult{Allowed: false, Reason: "permission denied"}
}
