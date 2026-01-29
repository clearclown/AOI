package tailscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aoi-protocol/aoi/internal/acl"
)

func TestACL_AddTagMapping(t *testing.T) {
	client := NewMockClient()
	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{})

	// Test valid tag mapping
	err := tsACL.AddTagMapping(TagPermissionMapping{
		Tag:        "tag:aoi-agent",
		Resources:  []string{"agents/*"},
		Permission: acl.PermissionWrite,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test invalid tag format
	err = tsACL.AddTagMapping(TagPermissionMapping{
		Tag:        "invalid-tag",
		Resources:  []string{"agents/*"},
		Permission: acl.PermissionWrite,
	})
	if err != ErrInvalidTagFormat {
		t.Errorf("expected ErrInvalidTagFormat, got %v", err)
	}

	// Test updating existing mapping
	err = tsACL.AddTagMapping(TagPermissionMapping{
		Tag:        "tag:aoi-agent",
		Resources:  []string{"agents/*", "queries/*"},
		Permission: acl.PermissionAdmin,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mappings := tsACL.GetTagMappings()
	if len(mappings) != 1 {
		t.Errorf("expected 1 mapping, got %d", len(mappings))
	}
	if len(mappings[0].Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(mappings[0].Resources))
	}
}

func TestACL_RemoveTagMapping(t *testing.T) {
	client := NewMockClient()
	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{Tag: "tag:aoi-agent", Resources: []string{"agents/*"}, Permission: acl.PermissionWrite},
		},
	})

	// Remove existing mapping
	err := tsACL.RemoveTagMapping("tag:aoi-agent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mappings := tsACL.GetTagMappings()
	if len(mappings) != 0 {
		t.Errorf("expected 0 mappings, got %d", len(mappings))
	}

	// Try to remove non-existent mapping
	err = tsACL.RemoveTagMapping("tag:non-existent")
	if err != ErrTagMappingNotFound {
		t.Errorf("expected ErrTagMappingNotFound, got %v", err)
	}
}

func TestACL_GetPermissionsForTags(t *testing.T) {
	client := NewMockClient()
	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-admin",
				Resources:  []string{"*"},
				Permission: acl.PermissionAdmin,
			},
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*", "queries/*"},
				Permission: acl.PermissionWrite,
			},
			{
				Tag:        "tag:aoi-reader",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionRead,
			},
		},
	})

	// Test single tag
	rules := tsACL.GetPermissionsForTags([]string{"tag:aoi-agent"})
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}

	// Test multiple tags (should get highest permission)
	rules = tsACL.GetPermissionsForTags([]string{"tag:aoi-agent", "tag:aoi-reader"})
	// Should have agents/* (Write from agent tag) and queries/* (Write from agent tag)
	foundAgents := false
	for _, rule := range rules {
		if rule.Resource == "agents/*" {
			foundAgents = true
			if rule.Permission != acl.PermissionWrite {
				t.Errorf("expected PermissionWrite for agents/*, got %v", rule.Permission)
			}
		}
	}
	if !foundAgents {
		t.Error("expected agents/* rule")
	}

	// Test admin tag
	rules = tsACL.GetPermissionsForTags([]string{"tag:aoi-admin"})
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Permission != acl.PermissionAdmin {
		t.Errorf("expected PermissionAdmin, got %v", rules[0].Permission)
	}

	// Test unknown tag
	rules = tsACL.GetPermissionsForTags([]string{"tag:unknown"})
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestACL_GetPermissionsForNode(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:aoi-agent"},
	})

	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionWrite,
			},
		},
	})

	ctx := context.Background()
	rules, err := tsACL.GetPermissionsForNode(ctx, "node-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
}

func TestACL_CheckPermissionForTags(t *testing.T) {
	client := NewMockClient()
	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionWrite,
			},
		},
		DefaultPermission: acl.PermissionNone,
	})

	tests := []struct {
		tags     []string
		resource string
		action   string
		allowed  bool
	}{
		{
			tags:     []string{"tag:aoi-agent"},
			resource: "agents/agent-1",
			action:   "read",
			allowed:  true,
		},
		{
			tags:     []string{"tag:aoi-agent"},
			resource: "agents/agent-1",
			action:   "write",
			allowed:  true,
		},
		{
			tags:     []string{"tag:aoi-agent"},
			resource: "agents/agent-1",
			action:   "admin",
			allowed:  false,
		},
		{
			tags:     []string{"tag:unknown"},
			resource: "agents/agent-1",
			action:   "read",
			allowed:  false,
		},
		{
			tags:     []string{"tag:aoi-agent"},
			resource: "tasks/task-1",
			action:   "read",
			allowed:  false,
		},
	}

	for i, tt := range tests {
		result := tsACL.CheckPermissionForTags(tt.tags, tt.resource, tt.action)
		if result.Allowed != tt.allowed {
			t.Errorf("test %d: CheckPermissionForTags(%v, %s, %s) = %v, expected %v",
				i, tt.tags, tt.resource, tt.action, result.Allowed, tt.allowed)
		}
	}
}

func TestACL_CheckPermissionForNode(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:aoi-agent"},
	})

	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionWrite,
			},
		},
	})

	ctx := context.Background()

	// Test allowed action
	result, err := tsACL.CheckPermissionForNode(ctx, "node-1", "agents/agent-1", "write")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("expected permission to be allowed")
	}

	// Test denied action
	result, err = tsACL.CheckPermissionForNode(ctx, "node-1", "tasks/task-1", "write")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected permission to be denied")
	}

	// Test non-existent node
	_, err = tsACL.CheckPermissionForNode(ctx, "non-existent", "agents/agent-1", "write")
	if err == nil {
		t.Error("expected error for non-existent node")
	}
}

func TestACL_SyncNodeACL(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:aoi-agent"},
	})

	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionWrite,
			},
		},
	})

	ctx := context.Background()
	err := tsACL.SyncNodeACL(ctx, "node-1", "agent-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify rules were added to ACL manager
	result := aclManager.CheckPermission("agent-1", "agents/*", "write")
	if !result.Allowed {
		t.Error("expected permission to be synced to ACL manager")
	}
}

func TestACL_DetectTagChanges(t *testing.T) {
	client := NewMockClient()
	client.AddPeer(&NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:aoi-agent"},
	})

	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{})

	ctx := context.Background()

	// First call should detect all nodes as changed (not in cache)
	changed, err := tsACL.DetectTagChanges(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(changed) != 1 {
		t.Errorf("expected 1 changed node, got %d", len(changed))
	}

	// Second call with same tags should detect no changes
	changed, err = tsACL.DetectTagChanges(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("expected 0 changed nodes, got %d", len(changed))
	}

	// Change tags and detect
	client.AddPeer(&NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:aoi-agent", "tag:production"},
	})

	changed, err = tsACL.DetectTagChanges(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(changed) != 1 {
		t.Errorf("expected 1 changed node, got %d", len(changed))
	}
}

func TestMatchResource(t *testing.T) {
	tests := []struct {
		pattern  string
		target   string
		expected bool
	}{
		{"agents/*", "agents/agent-1", true},
		{"agents/*", "agents/agent-1/tasks", false},
		{"agents/*", "tasks/task-1", false},
		{"*", "anything", true},
		{"*", "agents/agent-1", true},
		{"agents/agent-1", "agents/agent-1", true},
		{"agents/agent-1", "agents/agent-2", false},
	}

	for _, tt := range tests {
		result := matchResource(tt.pattern, tt.target)
		if result != tt.expected {
			t.Errorf("matchResource(%s, %s) = %v, expected %v",
				tt.pattern, tt.target, result, tt.expected)
		}
	}
}

func TestActionToPermission(t *testing.T) {
	tests := []struct {
		action   string
		expected acl.PermissionLevel
	}{
		{"read", acl.PermissionRead},
		{"write", acl.PermissionWrite},
		{"execute", acl.PermissionWrite},
		{"admin", acl.PermissionAdmin},
		{"unknown", acl.PermissionNone},
	}

	for _, tt := range tests {
		result := actionToPermission(tt.action)
		if result != tt.expected {
			t.Errorf("actionToPermission(%s) = %v, expected %v",
				tt.action, result, tt.expected)
		}
	}
}

func TestTagsEqual(t *testing.T) {
	tests := []struct {
		a        []string
		b        []string
		expected bool
	}{
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"b", "a"}, true},
		{[]string{"a"}, []string{"a", "b"}, false},
		{[]string{"a", "b"}, []string{"a"}, false},
		{[]string{}, []string{}, true},
		{nil, nil, true},
	}

	for i, tt := range tests {
		result := tagsEqual(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("test %d: tagsEqual(%v, %v) = %v, expected %v",
				i, tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestDefaultTagMappings(t *testing.T) {
	mappings := DefaultTagMappings()

	if len(mappings) != 4 {
		t.Errorf("expected 4 default mappings, got %d", len(mappings))
	}

	// Check admin mapping
	found := false
	for _, m := range mappings {
		if m.Tag == "tag:aoi-admin" {
			found = true
			if m.Permission != acl.PermissionAdmin {
				t.Errorf("expected admin permission for tag:aoi-admin")
			}
			if len(m.Resources) != 1 || m.Resources[0] != "*" {
				t.Errorf("expected wildcard resource for admin tag")
			}
		}
	}
	if !found {
		t.Error("expected tag:aoi-admin in default mappings")
	}
}

func TestACL_PermissionMiddleware(t *testing.T) {
	client := NewMockClient()
	aclManager := acl.NewAclManager()
	tsACL := NewACL(client, aclManager, ACLConfig{
		TagMappings: []TagPermissionMapping{
			{
				Tag:        "tag:aoi-agent",
				Resources:  []string{"agents/*"},
				Permission: acl.PermissionWrite,
			},
		},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := tsACL.PermissionMiddleware("agents/agent-1", "write")
	protectedHandler := middleware(handler)

	// Test without node info (should fail)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	// Test with node info but without permission (should fail)
	nodeInfo := &NodeInfo{
		ID:   "node-1",
		Tags: []string{"tag:unknown"},
	}
	ctx := context.WithValue(context.Background(), ContextKeyNodeInfo, nodeInfo)
	req = httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rr = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	// Test with permission (should pass)
	nodeInfo.Tags = []string{"tag:aoi-agent"}
	ctx = context.WithValue(context.Background(), ContextKeyNodeInfo, nodeInfo)
	req = httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rr = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}
