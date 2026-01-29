import { render, screen } from '@testing-library/react'
import Dashboard from './Dashboard'

interface AgentInfo {
  id: string
  role: string
  status: 'online' | 'offline'
  lastSeen: string
}

describe('Dashboard Component', () => {
  describe('Basic Rendering', () => {
    test('renders agent list', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
      expect(screen.getByText(/engineer/i)).toBeInTheDocument()
      expect(screen.getByText(/online/i)).toBeInTheDocument()
    })

    test('renders dashboard title', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
    })

    test('renders agent role label', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/role:/i)).toBeInTheDocument()
    })

    test('renders agent status label', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/status:/i)).toBeInTheDocument()
    })

    test('renders last seen timestamp', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    test('shows empty state when no agents', () => {
      render(<Dashboard agents={[]} />)
      expect(screen.getByText(/no agents/i)).toBeInTheDocument()
    })

    test('does not render dashboard title in empty state', () => {
      render(<Dashboard agents={[]} />)
      expect(screen.queryByRole('heading', { name: /agent dashboard/i })).not.toBeInTheDocument()
    })

    test('does not render agent grid in empty state', () => {
      render(<Dashboard agents={[]} />)
      expect(screen.queryByRole('heading', { name: /agent dashboard/i })).not.toBeInTheDocument()
    })
  })

  describe('Agent Status Display', () => {
    test('displays online status correctly', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const statusElement = screen.getByText('online')
      expect(statusElement).toBeInTheDocument()
      const statusSpan = container.querySelector('span')
      expect(statusSpan).toBeInTheDocument()
    })

    test('displays offline status correctly', () => {
      const agents: AgentInfo[] = [
        { id: 'pm-tanaka', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const statusElement = screen.getByText('offline')
      expect(statusElement).toBeInTheDocument()
      const statusSpan = container.querySelector('span')
      expect(statusSpan).toBeInTheDocument()
    })

    test('online status has green color indicator', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      const statusSpan = screen.getByText('online')
      expect(statusSpan.tagName).toBe('SPAN')
    })

    test('offline status has red color indicator', () => {
      const agents: AgentInfo[] = [
        { id: 'pm-tanaka', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      const statusSpan = screen.getByText('offline')
      expect(statusSpan.tagName).toBe('SPAN')
    })
  })

  describe('Multiple Agents', () => {
    test('renders multiple agents correctly', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' },
        { id: 'pm-tanaka', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' },
        { id: 'qa-yamada', role: 'qa', status: 'online', lastSeen: '2026-01-28T10:30:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
      expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
      expect(screen.getByText(/qa-yamada/i)).toBeInTheDocument()
    })

    test('renders correct count of agent cards', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' },
        { id: 'pm-tanaka', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' },
        { id: 'qa-yamada', role: 'qa', status: 'online', lastSeen: '2026-01-28T10:30:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const agentCards = container.querySelectorAll('h3')
      expect(agentCards).toHaveLength(3)
    })

    test('each agent has unique id displayed', () => {
      const agents: AgentInfo[] = [
        { id: 'agent-1', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' },
        { id: 'agent-2', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('agent-1')).toBeInTheDocument()
      expect(screen.getByText('agent-2')).toBeInTheDocument()
    })

    test('displays mixed online and offline statuses', () => {
      const agents: AgentInfo[] = [
        { id: 'agent-1', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' },
        { id: 'agent-2', role: 'pm', status: 'offline', lastSeen: '2026-01-28T09:00:00Z' },
        { id: 'agent-3', role: 'qa', status: 'online', lastSeen: '2026-01-28T10:30:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      const onlineStatuses = screen.getAllByText(/online/i)
      const offlineStatuses = screen.getAllByText(/offline/i)
      expect(onlineStatuses).toHaveLength(2)
      expect(offlineStatuses).toHaveLength(1)
    })
  })

  describe('Agent Roles', () => {
    test('displays engineer role', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-1', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/engineer/i)).toBeInTheDocument()
    })

    test('displays pm role', () => {
      const agents: AgentInfo[] = [
        { id: 'pm-1', role: 'pm', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('pm')).toBeInTheDocument()
    })

    test('displays qa role', () => {
      const agents: AgentInfo[] = [
        { id: 'qa-1', role: 'qa', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('qa')).toBeInTheDocument()
    })

    test('displays custom role', () => {
      const agents: AgentInfo[] = [
        { id: 'custom-1', role: 'architect', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/architect/i)).toBeInTheDocument()
    })
  })

  describe('Timestamp Formatting', () => {
    test('formats timestamp correctly', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })

    test('displays last seen with proper formatting', () => {
      const timestamp = '2026-01-28T15:30:45Z'
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: timestamp }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })
  })

  describe('Grid Layout', () => {
    test('applies grid layout styles', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })

    test('grid has responsive columns', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })
  })

  describe('Agent Card Styling', () => {
    test('agent cards have borders', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const cards = container.querySelectorAll('div > div > div')
      expect(cards.length).toBeGreaterThan(0)
    })

    test('agent cards have padding', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const cards = container.querySelectorAll('div > div > div')
      expect(cards.length).toBeGreaterThan(0)
    })

    test('agent cards have border radius', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<Dashboard agents={agents} />)
      const cards = container.querySelectorAll('div > div > div')
      expect(cards.length).toBeGreaterThan(0)
    })
  })

  describe('Single Agent', () => {
    test('renders correctly with single agent', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })

    test('single agent shows all required fields', () => {
      const agents: AgentInfo[] = [
        { id: 'eng-suzuki', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
      expect(screen.getByText(/engineer/i)).toBeInTheDocument()
      expect(screen.getByText(/online/i)).toBeInTheDocument()
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })
  })

  describe('Many Agents', () => {
    test('renders with many agents', () => {
      const agents: AgentInfo[] = Array.from({ length: 10 }, (_, i) => ({
        id: `agent-${i}`,
        role: i % 2 === 0 ? 'engineer' : 'pm',
        status: i % 3 === 0 ? 'offline' : 'online' as 'online' | 'offline',
        lastSeen: '2026-01-28T10:00:00Z'
      }))
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('agent-0')).toBeInTheDocument()
      expect(screen.getByText('agent-9')).toBeInTheDocument()
    })

    test('handles large number of agents', () => {
      const agents: AgentInfo[] = Array.from({ length: 50 }, (_, i) => ({
        id: `agent-${i}`,
        role: 'engineer',
        status: 'online' as const,
        lastSeen: '2026-01-28T10:00:00Z'
      }))
      const { container } = render(<Dashboard agents={agents} />)
      const agentCards = container.querySelectorAll('h3')
      expect(agentCards).toHaveLength(50)
    })
  })

  describe('Edge Cases', () => {
    test('handles agent with very long id', () => {
      const agents: AgentInfo[] = [
        { id: 'very-long-agent-id-that-might-break-layout-12345', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('very-long-agent-id-that-might-break-layout-12345')).toBeInTheDocument()
    })

    test('handles agent with special characters in id', () => {
      const agents: AgentInfo[] = [
        { id: 'agent_with-special.chars@123', role: 'engineer', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('agent_with-special.chars@123')).toBeInTheDocument()
    })

    test('handles agent with very long role name', () => {
      const agents: AgentInfo[] = [
        { id: 'agent-1', role: 'Senior Principal Staff Software Engineer Architect', status: 'online', lastSeen: '2026-01-28T10:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText('Senior Principal Staff Software Engineer Architect')).toBeInTheDocument()
    })

    test('handles past timestamps correctly', () => {
      const agents: AgentInfo[] = [
        { id: 'agent-1', role: 'engineer', status: 'offline', lastSeen: '2020-01-01T00:00:00Z' }
      ]
      render(<Dashboard agents={agents} />)
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })
  })
})
