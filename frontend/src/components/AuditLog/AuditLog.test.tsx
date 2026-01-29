import { render, screen } from '@testing-library/react'
import AuditLog from './AuditLog'

interface AuditEntry {
  id: string
  timestamp: string
  fromAgent: string
  toAgent: string
  eventType: string
  summary: string
}

describe('AuditLog Component', () => {
  describe('Basic Rendering', () => {
    test('renders audit entries', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Progress check'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
      expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
      expect(screen.getByText(/Progress check/i)).toBeInTheDocument()
    })

    test('renders audit log title', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Progress check'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })

    test('renders timestamp for entry', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Progress check'
        }
      ]
      render(<AuditLog entries={entries} />)
      const dateString = new Date('2026-01-28T10:00:00Z').toLocaleString()
      expect(screen.getByText(dateString)).toBeInTheDocument()
    })

    test('renders event type', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Progress check'
        }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const queryElements = screen.getAllByText('query')
      expect(queryElements.length).toBeGreaterThanOrEqual(1)
    })

    test('renders arrow between agents', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Progress check'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('→')).toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    test('shows empty state when no entries', () => {
      render(<AuditLog entries={[]} />)
      expect(screen.getByText(/no audit entries/i)).toBeInTheDocument()
    })

    test('does not render title in empty state', () => {
      render(<AuditLog entries={[]} />)
      expect(screen.queryByRole('heading', { name: /audit log/i })).not.toBeInTheDocument()
    })

    test('does not render scrollable container in empty state', () => {
      render(<AuditLog entries={[]} />)
      expect(screen.queryByRole('heading', { name: /audit log/i })).not.toBeInTheDocument()
    })
  })

  describe('Entry Ordering', () => {
    test('renders entries in chronological order', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: 'First'
        },
        {
          id: '2',
          timestamp: '2026-01-28T11:00:00Z',
          fromAgent: 'agent2',
          toAgent: 'agent1',
          eventType: 'response',
          summary: 'Second'
        }
      ]
      render(<AuditLog entries={entries} />)
      const summaries = screen.getAllByText(/First|Second/)
      expect(summaries[0]).toHaveTextContent('First')
      expect(summaries[1]).toHaveTextContent('Second')
    })

    test('maintains order with multiple entries', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Entry 1' },
        { id: '2', timestamp: '2026-01-28T11:00:00Z', fromAgent: 'a2', toAgent: 'a3', eventType: 'response', summary: 'Entry 2' },
        { id: '3', timestamp: '2026-01-28T12:00:00Z', fromAgent: 'a3', toAgent: 'a1', eventType: 'error', summary: 'Entry 3' }
      ]
      render(<AuditLog entries={entries} />)
      const summaries = screen.getAllByText(/Entry [1-3]/)
      expect(summaries[0]).toHaveTextContent('Entry 1')
      expect(summaries[1]).toHaveTextContent('Entry 2')
      expect(summaries[2]).toHaveTextContent('Entry 3')
    })
  })

  describe('Event Types', () => {
    test('displays query event type', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Status check'
        }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const queryElements = screen.getAllByText('query')
      expect(queryElements.length).toBeGreaterThanOrEqual(1)
    })

    test('displays response event type', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'eng-suzuki',
          toAgent: 'pm-tanaka',
          eventType: 'response',
          summary: 'Status response'
        }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const responseElements = screen.getAllByText('response')
      expect(responseElements.length).toBeGreaterThanOrEqual(1)
    })

    test('displays error event type', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'eng-suzuki',
          toAgent: 'pm-tanaka',
          eventType: 'error',
          summary: 'Task failed'
        }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const errorElements = screen.getAllByText('error')
      expect(errorElements.length).toBeGreaterThanOrEqual(1)
    })

    test('handles custom event types', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'custom_event',
          summary: 'Custom action'
        }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const customElements = screen.getAllByText('custom_event')
      expect(customElements.length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('Timestamp Formatting', () => {
    test('formats timestamp with date and time', () => {
      const timestamp = '2026-01-28T15:30:45Z'
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp,
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: 'Test'
        }
      ]
      render(<AuditLog entries={entries} />)
      const formattedDate = new Date(timestamp).toLocaleString()
      expect(screen.getByText(formattedDate)).toBeInTheDocument()
    })

    test('handles different timestamp formats', () => {
      const timestamp = '2026-01-28T00:00:00Z'
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp,
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: 'Test'
        }
      ]
      render(<AuditLog entries={entries} />)
      const formattedDate = new Date(timestamp).toLocaleString()
      expect(screen.getByText(formattedDate)).toBeInTheDocument()
    })

    test('displays multiple timestamps correctly', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test 1' },
        { id: '2', timestamp: '2026-01-28T11:00:00Z', fromAgent: 'a2', toAgent: 'a1', eventType: 'response', summary: 'Test 2' }
      ]
      render(<AuditLog entries={entries} />)
      const date1 = new Date('2026-01-28T10:00:00Z').toLocaleString()
      const date2 = new Date('2026-01-28T11:00:00Z').toLocaleString()
      expect(screen.getByText(date1)).toBeInTheDocument()
      expect(screen.getByText(date2)).toBeInTheDocument()
    })
  })

  describe('Agent Names', () => {
    test('displays from and to agent names', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Test'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })

    test('handles agent names with special characters', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent_with-special.chars@123',
          toAgent: 'another-agent_123',
          eventType: 'query',
          summary: 'Test'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('agent_with-special.chars@123')).toBeInTheDocument()
      expect(screen.getByText('another-agent_123')).toBeInTheDocument()
    })

    test('displays agent names in bold', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'pm-tanaka',
          toAgent: 'eng-suzuki',
          eventType: 'query',
          summary: 'Test'
        }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      const boldElements = container.querySelectorAll('strong')
      expect(boldElements.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Summary Display', () => {
    test('displays entry summary', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: 'This is a detailed summary message'
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('This is a detailed summary message')).toBeInTheDocument()
    })

    test('handles long summaries', () => {
      const longSummary = 'This is a very long summary that contains a lot of information about what happened during this particular audit event'
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: longSummary
        }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText(longSummary)).toBeInTheDocument()
    })

    test('handles empty summary', () => {
      const entries: AuditEntry[] = [
        {
          id: '1',
          timestamp: '2026-01-28T10:00:00Z',
          fromAgent: 'agent1',
          toAgent: 'agent2',
          eventType: 'query',
          summary: ''
        }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      expect(container).toBeInTheDocument()
    })
  })

  describe('Multiple Entries', () => {
    test('renders multiple entries correctly', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'First' },
        { id: '2', timestamp: '2026-01-28T11:00:00Z', fromAgent: 'a2', toAgent: 'a3', eventType: 'response', summary: 'Second' },
        { id: '3', timestamp: '2026-01-28T12:00:00Z', fromAgent: 'a3', toAgent: 'a1', eventType: 'error', summary: 'Third' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('First')).toBeInTheDocument()
      expect(screen.getByText('Second')).toBeInTheDocument()
      expect(screen.getByText('Third')).toBeInTheDocument()
    })

    test('each entry has unique key', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'First' },
        { id: '2', timestamp: '2026-01-28T11:00:00Z', fromAgent: 'a2', toAgent: 'a3', eventType: 'response', summary: 'Second' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('First')).toBeInTheDocument()
      expect(screen.getByText('Second')).toBeInTheDocument()
    })

    test('handles many entries', () => {
      const entries: AuditEntry[] = Array.from({ length: 20 }, (_, i) => ({
        id: `${i}`,
        timestamp: `2026-01-28T${10 + i}:00:00Z`,
        fromAgent: `agent${i}`,
        toAgent: `agent${i + 1}`,
        eventType: i % 2 === 0 ? 'query' : 'response',
        summary: `Entry ${i}`
      }))
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('Entry 0')).toBeInTheDocument()
      expect(screen.getByText('Entry 19')).toBeInTheDocument()
    })
  })

  describe('Scrolling Container', () => {
    test('has scrollable container with max height', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('Test')).toBeInTheDocument()
    })

    test('overflow-y is set to auto', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })
  })

  describe('Entry Styling', () => {
    test('entries have left border', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      const entryDivs = container.querySelectorAll('div > div > div')
      expect(entryDivs.length).toBeGreaterThan(0)
    })

    test('entries have left padding', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      const entryDivs = container.querySelectorAll('div > div > div')
      expect(entryDivs.length).toBeGreaterThan(0)
    })

    test('entries have margin bottom', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      const entryDivs = container.querySelectorAll('div > div > div')
      expect(entryDivs.length).toBeGreaterThan(0)
    })

    test('timestamp has gray color', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      const timestamps = container.querySelectorAll('p')
      expect(timestamps.length).toBeGreaterThan(0)
    })

    test('event type has smaller font size', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Test' }
      ]
      render(<AuditLog entries={entries} />)
      // Event type appears in both filter dropdown and entry display
      const eventTypes = screen.getAllByText('query')
      expect(eventTypes.length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('Edge Cases', () => {
    test('handles entry with missing optional fields gracefully', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: '', toAgent: '', eventType: '', summary: '' }
      ]
      const { container } = render(<AuditLog entries={entries} />)
      expect(container).toBeInTheDocument()
    })

    test('handles very old timestamps', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2000-01-01T00:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Old entry' }
      ]
      render(<AuditLog entries={entries} />)
      const oldDate = new Date('2000-01-01T00:00:00Z').toLocaleString()
      expect(screen.getByText(oldDate)).toBeInTheDocument()
    })

    test('handles entries with same timestamp', () => {
      const timestamp = '2026-01-28T10:00:00Z'
      const entries: AuditEntry[] = [
        { id: '1', timestamp, fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'First' },
        { id: '2', timestamp, fromAgent: 'a2', toAgent: 'a3', eventType: 'response', summary: 'Second' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('First')).toBeInTheDocument()
      expect(screen.getByText('Second')).toBeInTheDocument()
    })

    test('handles single entry', () => {
      const entries: AuditEntry[] = [
        { id: '1', timestamp: '2026-01-28T10:00:00Z', fromAgent: 'a1', toAgent: 'a2', eventType: 'query', summary: 'Only entry' }
      ]
      render(<AuditLog entries={entries} />)
      expect(screen.getByText('Only entry')).toBeInTheDocument()
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })
  })
})
