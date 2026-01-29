package identity

import (
	"errors"
	"sync"

	"github.com/aoi-protocol/aoi/pkg/aoi"
)

// AgentRegistry manages registered agents
type AgentRegistry struct {
	agents map[string]*aoi.AgentIdentity
	mu     sync.RWMutex
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*aoi.AgentIdentity),
	}
}

// Register adds an agent to the registry
func (r *AgentRegistry) Register(agent *aoi.AgentIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agent.ID] = agent
	return nil
}

// GetAgent retrieves an agent by ID
func (r *AgentRegistry) GetAgent(id string) (*aoi.AgentIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[id]
	if !exists {
		return nil, errors.New("agent not found")
	}

	return agent, nil
}

// Discover returns all registered agents
func (r *AgentRegistry) Discover() []*aoi.AgentIdentity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*aoi.AgentIdentity, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents
}

// UpdateStatus updates an agent's status
func (r *AgentRegistry) UpdateStatus(id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[id]
	if !exists {
		return errors.New("agent not found")
	}

	agent.Status = status
	return nil
}

// GetAgentByTailscaleNodeID retrieves an agent by its Tailscale node ID
func (r *AgentRegistry) GetAgentByTailscaleNodeID(nodeID string) (*aoi.AgentIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, agent := range r.agents {
		if agent.TailscaleNodeID == nodeID {
			return agent, nil
		}
	}

	return nil, errors.New("agent not found for Tailscale node ID")
}

// UpdateTailscaleNodeID updates an agent's Tailscale node ID
func (r *AgentRegistry) UpdateTailscaleNodeID(id string, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[id]
	if !exists {
		return errors.New("agent not found")
	}

	agent.TailscaleNodeID = nodeID
	return nil
}

// Unregister removes an agent from the registry
func (r *AgentRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; !exists {
		return errors.New("agent not found")
	}

	delete(r.agents, id)
	return nil
}
