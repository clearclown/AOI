package tailscale

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/aoi-protocol/aoi/internal/acl"
)

// ACL errors
var (
	ErrTagMappingNotFound = errors.New("tag mapping not found")
	ErrInvalidTagFormat   = errors.New("invalid tag format")
)

// TagPermissionMapping maps a Tailscale tag to AOI permissions
type TagPermissionMapping struct {
	Tag        string              // Tailscale tag (e.g., "tag:aoi-agent")
	Resources  []string            // Resources this tag can access
	Permission acl.PermissionLevel // Permission level for these resources
}

// ACLConfig holds ACL integration configuration
type ACLConfig struct {
	// TagMappings maps Tailscale tags to AOI permissions
	TagMappings []TagPermissionMapping
	// DefaultPermission is the permission level for unmapped tags
	DefaultPermission acl.PermissionLevel
	// SyncInterval is how often to sync with Tailscale (in seconds)
	SyncInterval int
}

// ACL provides Tailscale ACL integration with the AOI ACL manager
type ACL struct {
	client     Client
	aclManager *acl.AclManager
	config     ACLConfig
	mu         sync.RWMutex

	// Maps Tailscale node IDs to their current tags for change detection
	nodeTagCache map[string][]string
}

// NewACL creates a new Tailscale ACL integration
func NewACL(client Client, aclManager *acl.AclManager, config ACLConfig) *ACL {
	return &ACL{
		client:       client,
		aclManager:   aclManager,
		config:       config,
		nodeTagCache: make(map[string][]string),
	}
}

// GetTagMappings returns the current tag mappings
func (a *ACL) GetTagMappings() []TagPermissionMapping {
	a.mu.RLock()
	defer a.mu.RUnlock()

	mappings := make([]TagPermissionMapping, len(a.config.TagMappings))
	copy(mappings, a.config.TagMappings)
	return mappings
}

// AddTagMapping adds a new tag permission mapping
func (a *ACL) AddTagMapping(mapping TagPermissionMapping) error {
	if !strings.HasPrefix(mapping.Tag, "tag:") {
		return ErrInvalidTagFormat
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if mapping already exists
	for i, m := range a.config.TagMappings {
		if m.Tag == mapping.Tag {
			a.config.TagMappings[i] = mapping
			return nil
		}
	}

	a.config.TagMappings = append(a.config.TagMappings, mapping)
	return nil
}

// RemoveTagMapping removes a tag mapping
func (a *ACL) RemoveTagMapping(tag string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, m := range a.config.TagMappings {
		if m.Tag == tag {
			a.config.TagMappings = append(a.config.TagMappings[:i], a.config.TagMappings[i+1:]...)
			return nil
		}
	}

	return ErrTagMappingNotFound
}

// GetPermissionsForNode returns the AOI permissions for a Tailscale node
func (a *ACL) GetPermissionsForNode(ctx context.Context, nodeID string) ([]acl.AccessRule, error) {
	tags, err := a.client.GetNodeTags(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	return a.GetPermissionsForTags(tags), nil
}

// GetPermissionsForTags returns the AOI permissions for a set of Tailscale tags
func (a *ACL) GetPermissionsForTags(tags []string) []acl.AccessRule {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var rules []acl.AccessRule
	seenResources := make(map[string]acl.PermissionLevel)

	for _, tag := range tags {
		for _, mapping := range a.config.TagMappings {
			if mapping.Tag == tag {
				for _, resource := range mapping.Resources {
					// Track highest permission for each resource
					if current, ok := seenResources[resource]; !ok || mapping.Permission > current {
						seenResources[resource] = mapping.Permission
					}
				}
			}
		}
	}

	for resource, permission := range seenResources {
		rules = append(rules, acl.AccessRule{
			Resource:   resource,
			Permission: permission,
		})
	}

	return rules
}

// SyncNodeACL syncs ACL rules for a specific node to the ACL manager
func (a *ACL) SyncNodeACL(ctx context.Context, nodeID, agentID string) error {
	rules, err := a.GetPermissionsForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		rule.AgentID = agentID
		a.aclManager.AddRule(&rule)
	}

	return nil
}

// CheckPermissionForNode checks if a node has permission for a specific action
func (a *ACL) CheckPermissionForNode(ctx context.Context, nodeID, resource, action string) (acl.PermissionCheckResult, error) {
	tags, err := a.client.GetNodeTags(ctx, nodeID)
	if err != nil {
		return acl.PermissionCheckResult{Allowed: false, Reason: err.Error()}, err
	}

	return a.CheckPermissionForTags(tags, resource, action), nil
}

// CheckPermissionForTags checks if a set of tags has permission for an action
func (a *ACL) CheckPermissionForTags(tags []string, resource, action string) acl.PermissionCheckResult {
	rules := a.GetPermissionsForTags(tags)

	requiredPermission := actionToPermission(action)

	for _, rule := range rules {
		if matchResource(rule.Resource, resource) {
			if rule.Permission >= requiredPermission {
				return acl.PermissionCheckResult{Allowed: true, Reason: "permission granted via tag"}
			}
		}
	}

	// Check default permission
	if a.config.DefaultPermission >= requiredPermission {
		return acl.PermissionCheckResult{Allowed: true, Reason: "default permission granted"}
	}

	return acl.PermissionCheckResult{Allowed: false, Reason: "no matching tag permission"}
}

// SyncAllNodes syncs ACL rules for all known nodes
func (a *ACL) SyncAllNodes(ctx context.Context, nodeAgentMap map[string]string) error {
	for nodeID, agentID := range nodeAgentMap {
		if err := a.SyncNodeACL(ctx, nodeID, agentID); err != nil {
			// Log error but continue with other nodes
			continue
		}
	}
	return nil
}

// DetectTagChanges checks if any node's tags have changed
func (a *ACL) DetectTagChanges(ctx context.Context) ([]string, error) {
	var changedNodes []string

	peers, err := a.client.GetPeers(ctx)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for _, peer := range peers {
		cachedTags, exists := a.nodeTagCache[peer.ID]
		if !exists || !tagsEqual(cachedTags, peer.Tags) {
			changedNodes = append(changedNodes, peer.ID)
			a.nodeTagCache[peer.ID] = peer.Tags
		}
	}

	return changedNodes, nil
}

// actionToPermission converts an action string to a permission level
func actionToPermission(action string) acl.PermissionLevel {
	switch action {
	case "read":
		return acl.PermissionRead
	case "write", "execute":
		return acl.PermissionWrite
	case "admin":
		return acl.PermissionAdmin
	default:
		return acl.PermissionNone
	}
}

// matchResource checks if a resource pattern matches a target resource
func matchResource(pattern, target string) bool {
	// Exact match
	if pattern == target {
		return true
	}

	// Global wildcard
	if pattern == "*" {
		return true
	}

	// Wildcard match (e.g., "agents/*" matches "agents/agent-1" but not "agents/agent-1/tasks")
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if !strings.HasPrefix(target, prefix+"/") {
			return false
		}
		// Check if target has more path segments after the prefix
		remainder := strings.TrimPrefix(target, prefix+"/")
		// If remainder contains another slash, it's a deeper path - don't match
		return !strings.Contains(remainder, "/")
	}

	// Double wildcard match (e.g., "agents/**" matches "agents/agent-1/tasks")
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(target, prefix+"/")
	}

	return false
}

// tagsEqual checks if two tag slices are equal
func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	tagSet := make(map[string]bool)
	for _, tag := range a {
		tagSet[tag] = true
	}

	for _, tag := range b {
		if !tagSet[tag] {
			return false
		}
	}

	return true
}

// DefaultTagMappings returns sensible default tag mappings for AOI
func DefaultTagMappings() []TagPermissionMapping {
	return []TagPermissionMapping{
		{
			Tag:        "tag:aoi-admin",
			Resources:  []string{"*"},
			Permission: acl.PermissionAdmin,
		},
		{
			Tag:        "tag:aoi-agent",
			Resources:  []string{"agents/*", "queries/*", "tasks/*"},
			Permission: acl.PermissionWrite,
		},
		{
			Tag:        "tag:aoi-reader",
			Resources:  []string{"agents/*", "queries/*"},
			Permission: acl.PermissionRead,
		},
		{
			Tag:        "tag:aoi-executor",
			Resources:  []string{"tasks/*"},
			Permission: acl.PermissionWrite,
		},
	}
}

// PermissionMiddleware returns HTTP middleware that checks Tailscale-based permissions
func (a *ACL) PermissionMiddleware(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nodeInfo := GetNodeInfoFromContext(r.Context())
			if nodeInfo == nil {
				http.Error(w, "Unauthorized: no node info", http.StatusUnauthorized)
				return
			}

			result := a.CheckPermissionForTags(nodeInfo.Tags, resource, action)
			if !result.Allowed {
				http.Error(w, "Forbidden: "+result.Reason, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
