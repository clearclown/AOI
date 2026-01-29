# Frontend Player Instructions - AOI Protocol

## Your Mission
Create a working React + TypeScript frontend for the AOI (Agent Operational Interconnect) protocol following TDD principles.

## Working Directory
`/home/ablaze/Projects/AOI/frontend`

## Initialization

### Step 1: Create Vite React App
```bash
cd /home/ablaze/Projects/AOI
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
```

### Step 2: Install Testing Dependencies
```bash
npm install -D vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom
```

### Step 3: Configure Vitest
Create `vite.config.ts`:
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
```

Create `src/test/setup.ts`:
```typescript
import '@testing-library/jest-dom'
```

### Step 4: Update package.json Scripts
```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "test": "vitest",
    "preview": "vite preview"
  }
}
```

## TDD Protocol (MANDATORY)
For EACH component:
1. **RED**: Write failing test first
2. **GREEN**: Write minimal JSX to pass test
3. **REFACTOR**: Clean up while tests pass

Example workflow:
```bash
# 1. Write test
cat > src/components/ApprovalUI/ApprovalUI.test.tsx << 'EOF'
import { render, screen } from '@testing-library/react'
import ApprovalUI from './ApprovalUI'

test('renders approval pending message', () => {
  render(<ApprovalUI />)
  expect(screen.getByText(/approval pending/i)).toBeInTheDocument()
})
EOF

# 2. Run test (should fail)
npm test -- --run

# 3. Implement component
cat > src/components/ApprovalUI/ApprovalUI.tsx << 'EOF'
export default function ApprovalUI() {
  return <div>Approval Pending</div>
}
EOF

# 4. Run test (should pass)
npm test -- --run
```

## Required Structure
```
frontend/
├── src/
│   ├── components/
│   │   ├── ApprovalUI/
│   │   │   ├── ApprovalUI.tsx
│   │   │   └── ApprovalUI.test.tsx
│   │   ├── AuditLog/
│   │   │   ├── AuditLog.tsx
│   │   │   └── AuditLog.test.tsx
│   │   └── Dashboard/
│   │       ├── Dashboard.tsx
│   │       └── Dashboard.test.tsx
│   ├── hooks/
│   │   └── useAgents.ts
│   ├── services/
│   │   └── api.ts
│   ├── test/
│   │   └── setup.ts
│   ├── App.tsx
│   ├── App.test.tsx
│   └── main.tsx
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## Components to Implement

### 1. ApprovalUI Component
**Purpose**: Display pending approval requests and allow approve/deny

**Interface:**
```typescript
interface ApprovalRequest {
  id: string
  requester: string
  taskType: string
  params: Record<string, unknown>
  timestamp: string
}
```

**Test file (src/components/ApprovalUI/ApprovalUI.test.tsx):**
```typescript
import { render, screen, fireEvent } from '@testing-library/react'
import ApprovalUI from './ApprovalUI'

test('renders approval request list', () => {
  const requests = [
    { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
  ]
  render(<ApprovalUI requests={requests} />)
  expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
  expect(screen.getByText(/run_tests/i)).toBeInTheDocument()
})

test('approve button calls onApprove', () => {
  const onApprove = vi.fn()
  const requests = [
    { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
  ]
  render(<ApprovalUI requests={requests} onApprove={onApprove} />)

  fireEvent.click(screen.getByRole('button', { name: /approve/i }))
  expect(onApprove).toHaveBeenCalledWith('1')
})

test('deny button calls onDeny', () => {
  const onDeny = vi.fn()
  const requests = [
    { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
  ]
  render(<ApprovalUI requests={requests} onDeny={onDeny} />)

  fireEvent.click(screen.getByRole('button', { name: /deny/i }))
  expect(onDeny).toHaveBeenCalledWith('1')
})

test('shows empty state when no requests', () => {
  render(<ApprovalUI requests={[]} />)
  expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
})
```

**Implementation (src/components/ApprovalUI/ApprovalUI.tsx):**
```typescript
interface ApprovalRequest {
  id: string
  requester: string
  taskType: string
  params: Record<string, unknown>
  timestamp: string
}

interface ApprovalUIProps {
  requests: ApprovalRequest[]
  onApprove?: (id: string) => void
  onDeny?: (id: string) => void
}

export default function ApprovalUI({ requests, onApprove, onDeny }: ApprovalUIProps) {
  if (requests.length === 0) {
    return <div>No pending approvals</div>
  }

  return (
    <div>
      <h2>Pending Approvals</h2>
      {requests.map(req => (
        <div key={req.id} style={{ border: '1px solid #ccc', padding: '10px', margin: '10px 0' }}>
          <p><strong>Requester:</strong> {req.requester}</p>
          <p><strong>Task:</strong> {req.taskType}</p>
          <p><strong>Time:</strong> {new Date(req.timestamp).toLocaleString()}</p>
          <button onClick={() => onApprove?.(req.id)}>Approve</button>
          <button onClick={() => onDeny?.(req.id)}>Deny</button>
        </div>
      ))}
    </div>
  )
}
```

### 2. AuditLog Component
**Purpose**: Display chronological timeline of agent interactions

**Interface:**
```typescript
interface AuditEntry {
  id: string
  timestamp: string
  fromAgent: string
  toAgent: string
  eventType: string
  summary: string
}
```

**Test file (src/components/AuditLog/AuditLog.test.tsx):**
```typescript
import { render, screen } from '@testing-library/react'
import AuditLog from './AuditLog'

test('renders audit entries', () => {
  const entries = [
    { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'pm-tanaka', toAgent: 'eng-suzuki', eventType: 'query', summary: 'Progress check' }
  ]
  render(<AuditLog entries={entries} />)
  expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
  expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
  expect(screen.getByText(/Progress check/i)).toBeInTheDocument()
})

test('shows empty state when no entries', () => {
  render(<AuditLog entries={[]} />)
  expect(screen.getByText(/no audit entries/i)).toBeInTheDocument()
})

test('renders entries in chronological order', () => {
  const entries = [
    { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'agent1', toAgent: 'agent2', eventType: 'query', summary: 'First' },
    { id: '2', timestamp: '2026-01-28T11:00:00Z', fromAgent: 'agent2', toAgent: 'agent1', eventType: 'response', summary: 'Second' }
  ]
  render(<AuditLog entries={entries} />)
  const summaries = screen.getAllByText(/First|Second/)
  expect(summaries[0]).toHaveTextContent('First')
  expect(summaries[1]).toHaveTextContent('Second')
})
```

**Implementation (src/components/AuditLog/AuditLog.tsx):**
```typescript
interface AuditEntry {
  id: string
  timestamp: string
  fromAgent: string
  toAgent: string
  eventType: string
  summary: string
}

interface AuditLogProps {
  entries: AuditEntry[]
}

export default function AuditLog({ entries }: AuditLogProps) {
  if (entries.length === 0) {
    return <div>No audit entries</div>
  }

  return (
    <div>
      <h2>Audit Log</h2>
      <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
        {entries.map(entry => (
          <div key={entry.id} style={{ borderLeft: '3px solid #007bff', paddingLeft: '10px', marginBottom: '15px' }}>
            <p style={{ fontSize: '0.9em', color: '#666' }}>{new Date(entry.timestamp).toLocaleString()}</p>
            <p><strong>{entry.fromAgent}</strong> → <strong>{entry.toAgent}</strong></p>
            <p>{entry.summary}</p>
            <span style={{ fontSize: '0.8em', color: '#999' }}>{entry.eventType}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
```

### 3. Dashboard Component
**Purpose**: Display agent status and connection info

**Interface:**
```typescript
interface AgentInfo {
  id: string
  role: string
  status: 'online' | 'offline'
  lastSeen: string
}
```

**Test file (src/components/Dashboard/Dashboard.test.tsx):**
```typescript
import { render, screen } from '@testing-library/react'
import Dashboard from './Dashboard'

test('renders agent list', () => {
  const agents = [
    { id: 'eng-suzuki', role: 'engineer', status: 'online' as const, lastSeen: '2026-01-28T10:00:00Z' }
  ]
  render(<Dashboard agents={agents} />)
  expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
  expect(screen.getByText(/engineer/i)).toBeInTheDocument()
  expect(screen.getByText(/online/i)).toBeInTheDocument()
})

test('shows empty state when no agents', () => {
  render(<Dashboard agents={[]} />)
  expect(screen.getByText(/no agents/i)).toBeInTheDocument()
})

test('displays offline status correctly', () => {
  const agents = [
    { id: 'pm-tanaka', role: 'pm', status: 'offline' as const, lastSeen: '2026-01-28T09:00:00Z' }
  ]
  render(<Dashboard agents={agents} />)
  expect(screen.getByText(/offline/i)).toBeInTheDocument()
})
```

**Implementation (src/components/Dashboard/Dashboard.tsx):**
```typescript
interface AgentInfo {
  id: string
  role: string
  status: 'online' | 'offline'
  lastSeen: string
}

interface DashboardProps {
  agents: AgentInfo[]
}

export default function Dashboard({ agents }: DashboardProps) {
  if (agents.length === 0) {
    return <div>No agents available</div>
  }

  return (
    <div>
      <h2>Agent Dashboard</h2>
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
```

### 4. Main App Component
**Test file (src/App.test.tsx):**
```typescript
import { render, screen } from '@testing-library/react'
import App from './App'

test('renders app title', () => {
  render(<App />)
  expect(screen.getByText(/AOI Protocol/i)).toBeInTheDocument()
})

test('renders all main sections', () => {
  render(<App />)
  expect(screen.getByText(/Dashboard/i)).toBeInTheDocument()
  expect(screen.getByText(/Audit Log/i)).toBeInTheDocument()
  expect(screen.getByText(/Approvals/i)).toBeInTheDocument()
})
```

**Implementation (src/App.tsx):**
```typescript
import { useState } from 'react'
import Dashboard from './components/Dashboard/Dashboard'
import AuditLog from './components/AuditLog/AuditLog'
import ApprovalUI from './components/ApprovalUI/ApprovalUI'
import './App.css'

function App() {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'audit' | 'approvals'>('dashboard')

  // Mock data
  const agents = [
    { id: 'eng-local', role: 'engineer', status: 'online' as const, lastSeen: new Date().toISOString() }
  ]

  const auditEntries = [
    { id: '1', timestamp: new Date().toISOString(), fromAgent: 'pm-tanaka', toAgent: 'eng-suzuki', eventType: 'query', summary: 'Status check' }
  ]

  const approvalRequests = []

  return (
    <div style={{ padding: '20px', fontFamily: 'system-ui' }}>
      <h1>AOI Protocol Dashboard</h1>

      <nav style={{ marginBottom: '20px' }}>
        <button onClick={() => setActiveTab('dashboard')} style={{ marginRight: '10px' }}>Dashboard</button>
        <button onClick={() => setActiveTab('audit')} style={{ marginRight: '10px' }}>Audit Log</button>
        <button onClick={() => setActiveTab('approvals')}>Approvals</button>
      </nav>

      {activeTab === 'dashboard' && <Dashboard agents={agents} />}
      {activeTab === 'audit' && <AuditLog entries={auditEntries} />}
      {activeTab === 'approvals' && <ApprovalUI requests={approvalRequests} />}
    </div>
  )
}

export default App
```

## Quality Gates (YOU MUST PASS)

### Gate 1: All Tests Pass
```bash
cd /home/ablaze/Projects/AOI/frontend
npm test -- --run
```
**Required**: Minimum 3 test files, all tests passing

### Gate 2: Build Succeeds
```bash
cd /home/ablaze/Projects/AOI/frontend
npm run build
```
**Required**: No build errors, dist/ folder created

### Gate 3: Dev Server Runs
```bash
npm run dev
# Should start on http://localhost:5173
```

## Completion Criteria
- [ ] Vite project initialized
- [ ] Testing configured (vitest)
- [ ] At least 3 test files (*.test.tsx)
- [ ] All tests pass
- [ ] Build succeeds
- [ ] All components render correctly

## Tips
- Start with simplest component (Dashboard)
- Use inline styles for prototype (no CSS files needed)
- Mock data is fine for prototype
- Keep components simple and functional
- Run tests frequently: `npm test -- --run`

## When You're Done
Respond with:
```
✅ Frontend Implementation Complete

Test Results:
- ApprovalUI.test.tsx: PASS (4 tests)
- AuditLog.test.tsx: PASS (3 tests)
- Dashboard.test.tsx: PASS (3 tests)
- App.test.tsx: PASS (2 tests)

Total: 4 test files, 12 tests, all passing

Build: SUCCESS (dist/ folder created)
Dev Server: READY (http://localhost:5173)
```
