import { useState, useEffect } from 'react'

interface AgentInfo {
  id: string
  role: string
  status: 'online' | 'offline'
  lastSeen: string
}

interface DashboardProps {
  agents: AgentInfo[]
  onRefresh?: () => void
}

export default function Dashboard({ agents, onRefresh }: DashboardProps) {
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date())

  useEffect(() => {
    setLastUpdated(new Date())
  }, [agents])

  if (agents.length === 0) {
    return <div>No agents available</div>
  }

  const handleRefresh = () => {
    if (onRefresh) {
      onRefresh()
    }
    setLastUpdated(new Date())
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '15px' }}>
        <h2>Agent Dashboard</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
          <span style={{ fontSize: '0.9em', color: '#666' }}>
            Last updated: {lastUpdated.toLocaleTimeString()}
          </span>
          <button onClick={handleRefresh} style={{ cursor: 'pointer' }}>
            Refresh
          </button>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '15px' }}>
        {agents.map(agent => (
          <div key={agent.id} style={{ border: '1px solid #ddd', padding: '15px', borderRadius: '8px' }}>
            <h3>{agent.id}</h3>
            <p><strong>Role:</strong> {agent.role}</p>
            <p>
              <strong>Status:</strong>{' '}
              <span style={{ color: agent.status === 'online' ? 'green' : 'red' }}>
                {agent.status}
              </span>
            </p>
            <p style={{ fontSize: '0.9em', color: '#666' }}>
              Last seen: {new Date(agent.lastSeen).toLocaleString()}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}
