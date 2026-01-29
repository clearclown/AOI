import { useState, useEffect } from 'react'
import Dashboard from './components/Dashboard/Dashboard'
import AuditLog from './components/AuditLog/AuditLog'
import ApprovalUI from './components/ApprovalUI/ApprovalUI'
import { useAgents } from './hooks/useAgents'
import { useAuditLog } from './hooks/useAuditLog'
import { apiClient } from './services/api'
import './App.css'

// Local types matching component expectations
interface AgentInfo {
  id: string
  role: string
  status: 'online' | 'offline'
  lastSeen: string
}

interface AuditEntryLocal {
  id: string
  timestamp: string
  fromAgent: string
  toAgent: string
  eventType: string
  summary: string
}

function App() {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'audit' | 'approvals'>('dashboard')
  const [isConnected, setIsConnected] = useState<boolean | null>(null)
  const approvalRequests: never[] = []

  // Fetch data from API with fallback to mock data
  const { agents: apiAgents, error: agentsError } = useAgents()
  const { entries: apiEntries, error: auditError } = useAuditLog()

  // Mock data fallback
  const mockAgents: AgentInfo[] = [
    { id: 'eng-local', role: 'engineer', status: 'online', lastSeen: new Date().toISOString() }
  ]

  const mockAuditEntries: AuditEntryLocal[] = [
    {
      id: '1',
      timestamp: new Date().toISOString(),
      fromAgent: 'pm-tanaka',
      toAgent: 'eng-suzuki',
      eventType: 'query',
      summary: 'Status check'
    }
  ]

  // Check backend connection status
  useEffect(() => {
    const checkConnection = async () => {
      try {
        await apiClient.getHealth()
        setIsConnected(true)
      } catch {
        setIsConnected(false)
      }
    }

    checkConnection()
    const interval = setInterval(checkConnection, 30000) // Check every 30 seconds

    return () => clearInterval(interval)
  }, [])

  // Use API data if available and connected, otherwise fallback to mock data
  const agents: AgentInfo[] = isConnected && !agentsError && apiAgents.length > 0
    ? apiAgents.map(a => ({ id: a.id, role: a.role, status: a.status === 'busy' ? 'offline' : a.status, lastSeen: a.lastSeen }))
    : mockAgents

  const auditEntries: AuditEntryLocal[] = isConnected && !auditError && apiEntries.length > 0
    ? apiEntries.map(e => ({ id: e.id, timestamp: e.timestamp, fromAgent: e.from, toAgent: e.to, eventType: e.eventType, summary: e.summary }))
    : mockAuditEntries

  const handleApprove = (id: string) => {
    console.log('Approved:', id)
    // TODO: Call API when backend endpoint is available
  }

  const handleDeny = (id: string) => {
    console.log('Denied:', id)
    // TODO: Call API when backend endpoint is available
  }

  return (
    <div style={{ padding: '20px', fontFamily: 'system-ui' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>AOI Protocol Dashboard</h1>
        <div style={{ fontSize: '14px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span
            style={{
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              backgroundColor: isConnected === null ? '#999' : isConnected ? '#4caf50' : '#f44336',
              display: 'inline-block'
            }}
          />
          <span>
            {isConnected === null ? 'Checking...' : isConnected ? 'Connected' : 'Disconnected (using mock data)'}
          </span>
        </div>
      </div>

      <nav style={{ marginBottom: '20px' }}>
        <button onClick={() => setActiveTab('dashboard')} style={{ marginRight: '10px' }}>Dashboard</button>
        <button onClick={() => setActiveTab('audit')} style={{ marginRight: '10px' }}>Audit Log</button>
        <button onClick={() => setActiveTab('approvals')}>Approvals</button>
      </nav>

      {activeTab === 'dashboard' && <Dashboard agents={agents} />}
      {activeTab === 'audit' && <AuditLog entries={auditEntries} />}
      {activeTab === 'approvals' && (
        <ApprovalUI
          requests={approvalRequests}
          onApprove={handleApprove}
          onDeny={handleDeny}
        />
      )}
    </div>
  )
}

export default App
