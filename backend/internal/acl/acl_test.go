package acl

import (
	"sync"
	"testing"
)

func TestAclManager_CheckPermission_Allow(t *testing.T) {
	acl := NewAclManager()

	// Add a rule: agent-1 can read repo-1
	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	result := acl.CheckPermission("agent-1", "repo-1", "read")
	if !result.Allowed {
		t.Error("Expected permission to be allowed")
	}
}

func TestAclManager_CheckPermission_Deny(t *testing.T) {
	acl := NewAclManager()

	// No rules, should deny by default
	result := acl.CheckPermission("agent-1", "repo-1", "read")
	if result.Allowed {
		t.Error("Expected permission to be denied")
	}
}

func TestAclManager_CheckPermission_WriteRequiresWritePermission(t *testing.T) {
	acl := NewAclManager()

	// Add read permission only
	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	// Try to write
	result := acl.CheckPermission("agent-1", "repo-1", "write")
	if result.Allowed {
		t.Error("Expected write to be denied with only read permission")
	}
}

func TestAclManager_CheckPermission_AdminAllowsAll(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionAdmin,
	})

	readResult := acl.CheckPermission("agent-1", "repo-1", "read")
	writeResult := acl.CheckPermission("agent-1", "repo-1", "write")

	if !readResult.Allowed || !writeResult.Allowed {
		t.Error("Expected admin permission to allow all actions")
	}
}

func TestAclManager_CheckPermission_ExecuteAction(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionWrite,
	})

	result := acl.CheckPermission("agent-1", "repo-1", "execute")
	if !result.Allowed {
		t.Error("Expected execute to be allowed with write permission")
	}
}

func TestAclManager_CheckPermission_UnknownAction(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionAdmin,
	})

	result := acl.CheckPermission("agent-1", "repo-1", "unknown-action")
	if result.Allowed {
		t.Error("Expected unknown action to be denied")
	}
}

func TestAclManager_CheckPermission_DifferentAgents(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	// Agent-2 should not have access
	result := acl.CheckPermission("agent-2", "repo-1", "read")
	if result.Allowed {
		t.Error("Expected agent-2 to be denied access")
	}
}

func TestAclManager_CheckPermission_DifferentResources(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	// Different resource should be denied
	result := acl.CheckPermission("agent-1", "repo-2", "read")
	if result.Allowed {
		t.Error("Expected access to repo-2 to be denied")
	}
}

func TestAclManager_CheckPermission_MultipleRules(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-2",
		Permission: PermissionWrite,
	})

	// Check both resources
	result1 := acl.CheckPermission("agent-1", "repo-1", "read")
	result2 := acl.CheckPermission("agent-1", "repo-2", "write")

	if !result1.Allowed || !result2.Allowed {
		t.Error("Expected both permissions to be allowed")
	}
}

func TestAclManager_CheckPermission_FirstMatchingRule(t *testing.T) {
	acl := NewAclManager()

	// Add multiple rules for same agent/resource
	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionWrite,
	})

	// Should match first rule (read only)
	result := acl.CheckPermission("agent-1", "repo-1", "read")
	if !result.Allowed {
		t.Error("Expected read to be allowed")
	}

	// Write should also be allowed due to second rule
	writeResult := acl.CheckPermission("agent-1", "repo-1", "write")
	if !writeResult.Allowed {
		t.Error("Expected write to be allowed by second rule")
	}
}

func TestAclManager_CheckPermission_EmptyAgentID(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	result := acl.CheckPermission("", "repo-1", "read")
	if !result.Allowed {
		t.Error("Expected empty agent ID to match")
	}
}

func TestAclManager_CheckPermission_EmptyResource(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "",
		Permission: PermissionRead,
	})

	result := acl.CheckPermission("agent-1", "", "read")
	if !result.Allowed {
		t.Error("Expected empty resource to match")
	}
}

func TestAclManager_CheckPermission_NoPermission(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionNone,
	})

	result := acl.CheckPermission("agent-1", "repo-1", "read")
	if result.Allowed {
		t.Error("Expected PermissionNone to deny access")
	}
}

func TestAclManager_AddRule_Multiple(t *testing.T) {
	acl := NewAclManager()

	for i := 0; i < 10; i++ {
		acl.AddRule(&AccessRule{
			AgentID:    "agent-1",
			Resource:   "repo-" + string(rune('0'+i)),
			Permission: PermissionRead,
		})
	}

	// Check one of them
	result := acl.CheckPermission("agent-1", "repo-5", "read")
	if !result.Allowed {
		t.Error("Expected permission to be allowed for repo-5")
	}
}

func TestAclManager_PermissionLevels(t *testing.T) {
	tests := []struct {
		name       string
		permission PermissionLevel
		action     string
		expected   bool
	}{
		{"None-Read", PermissionNone, "read", false},
		{"Read-Read", PermissionRead, "read", true},
		{"Read-Write", PermissionRead, "write", false},
		{"Write-Read", PermissionWrite, "read", true},
		{"Write-Write", PermissionWrite, "write", true},
		{"Write-Execute", PermissionWrite, "execute", true},
		{"Admin-Read", PermissionAdmin, "read", true},
		{"Admin-Write", PermissionAdmin, "write", true},
		{"Admin-Execute", PermissionAdmin, "execute", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl := NewAclManager()
			acl.AddRule(&AccessRule{
				AgentID:    "agent-1",
				Resource:   "resource-1",
				Permission: tt.permission,
			})

			result := acl.CheckPermission("agent-1", "resource-1", tt.action)
			if result.Allowed != tt.expected {
				t.Errorf("Expected %v for %s with %d permission, got %v",
					tt.expected, tt.action, tt.permission, result.Allowed)
			}
		})
	}
}

func TestAclManager_CheckPermission_Reason(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	allowResult := acl.CheckPermission("agent-1", "repo-1", "read")
	if allowResult.Reason != "permission granted" {
		t.Errorf("Expected reason 'permission granted', got '%s'", allowResult.Reason)
	}

	denyResult := acl.CheckPermission("agent-2", "repo-1", "read")
	if denyResult.Reason != "permission denied" {
		t.Errorf("Expected reason 'permission denied', got '%s'", denyResult.Reason)
	}
}

func TestAclManager_ConcurrentAddRule(t *testing.T) {
	acl := NewAclManager()
	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			acl.AddRule(&AccessRule{
				AgentID:    "agent-" + string(rune('0'+id%10)),
				Resource:   "repo-" + string(rune('0'+id%10)),
				Permission: PermissionRead,
			})
		}(i)
	}

	wg.Wait()

	// Verify some rules work
	result := acl.CheckPermission("agent-5", "repo-5", "read")
	if !result.Allowed {
		t.Error("Expected permission to be allowed after concurrent rule addition")
	}
}

func TestAclManager_ConcurrentCheckPermission(t *testing.T) {
	acl := NewAclManager()

	acl.AddRule(&AccessRule{
		AgentID:    "agent-1",
		Resource:   "repo-1",
		Permission: PermissionRead,
	})

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			result := acl.CheckPermission("agent-1", "repo-1", "read")
			if !result.Allowed {
				t.Error("Expected permission to be allowed in concurrent check")
			}
		}()
	}

	wg.Wait()
}

func TestAclManager_ConcurrentReadWrite(t *testing.T) {
	acl := NewAclManager()

	numReaders := 50
	numWriters := 50

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		go func(id int) {
			defer wg.Done()
			_ = acl.CheckPermission("agent-1", "repo-1", "read")
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			acl.AddRule(&AccessRule{
				AgentID:    "agent-" + string(rune('0'+id%10)),
				Resource:   "repo-" + string(rune('0'+id%10)),
				Permission: PermissionRead,
			})
		}(i)
	}

	wg.Wait()
}
