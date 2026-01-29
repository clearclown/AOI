import { render, screen, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import ApprovalUI from './ApprovalUI'

interface ApprovalRequest {
  id: string
  requester: string
  taskType: string
  params: Record<string, unknown>
  timestamp: string
}

describe('ApprovalUI Component', () => {
  describe('Basic Rendering', () => {
    test('renders approval request list', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
      expect(screen.getByText(/run_tests/i)).toBeInTheDocument()
    })

    test('renders pending approvals title', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByRole('heading', { name: /pending approvals/i })).toBeInTheDocument()
    })

    test('renders requester label', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText(/requester:/i)).toBeInTheDocument()
    })

    test('renders task label', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText(/task:/i)).toBeInTheDocument()
    })

    test('renders time label', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText(/time:/i)).toBeInTheDocument()
    })
  })

  describe('Button Interactions', () => {
    test('approve button calls onApprove', () => {
      const onApprove = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} />)

      const approveButton = screen.getAllByRole('button', { name: /approve/i })[0]
      fireEvent.click(approveButton)
      expect(onApprove).toHaveBeenCalledWith('1')
    })

    test('deny button calls onDeny', () => {
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onDeny={onDeny} />)

      const denyButton = screen.getAllByRole('button', { name: /deny/i })[0]
      fireEvent.click(denyButton)
      expect(onDeny).toHaveBeenCalledWith('1', 'No reason provided')
    })

    test('approve button calls with correct request id', () => {
      const onApprove = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: 'req-123', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} />)

      const approveButton = screen.getAllByRole('button', { name: /approve/i })[0]
      fireEvent.click(approveButton)
      expect(onApprove).toHaveBeenCalledWith('req-123')
    })

    test('deny button calls with correct request id', () => {
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: 'req-456', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onDeny={onDeny} />)

      const denyButton = screen.getAllByRole('button', { name: /deny/i })[0]
      fireEvent.click(denyButton)
      expect(onDeny).toHaveBeenCalledWith('req-456', 'No reason provided')
    })

    test('approve button can be clicked multiple times', () => {
      const onApprove = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} />)

      const approveButton = screen.getAllByRole('button', { name: /approve/i })[0]
      fireEvent.click(approveButton)
      fireEvent.click(approveButton)
      expect(onApprove).toHaveBeenCalledTimes(2)
    })

    test('deny button can be clicked multiple times', () => {
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onDeny={onDeny} />)

      const denyButton = screen.getAllByRole('button', { name: /deny/i })[0]
      fireEvent.click(denyButton)
      fireEvent.click(denyButton)
      expect(onDeny).toHaveBeenCalledTimes(2)
    })

    test('works without onApprove callback', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)

      const approveButton = screen.getByRole('button', { name: /approve/i })
      expect(() => fireEvent.click(approveButton)).not.toThrow()
    })

    test('works without onDeny callback', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)

      const denyButton = screen.getByRole('button', { name: /deny/i })
      expect(() => fireEvent.click(denyButton)).not.toThrow()
    })
  })

  describe('Empty State', () => {
    test('shows empty state when no requests', () => {
      render(<ApprovalUI requests={[]} />)
      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })

    test('does not render title in empty state', () => {
      render(<ApprovalUI requests={[]} />)
      expect(screen.queryByRole('heading', { name: /pending approvals/i })).not.toBeInTheDocument()
    })

    test('does not render buttons in empty state', () => {
      render(<ApprovalUI requests={[]} />)
      expect(screen.queryByRole('button', { name: /approve/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /deny/i })).not.toBeInTheDocument()
    })
  })

  describe('Multiple Requests', () => {
    test('renders multiple approval requests', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T11:00:00Z' },
        { id: '3', requester: 'qa-yamada', taskType: 'rollback', params: {}, timestamp: '2026-01-28T12:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
      expect(screen.getByText('qa-yamada')).toBeInTheDocument()
    })

    test('each request has approve and deny buttons', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T11:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      const approveButtons = screen.getAllByRole('button', { name: /approve/i })
      const denyButtons = screen.getAllByRole('button', { name: /deny/i })
      expect(approveButtons).toHaveLength(2)
      expect(denyButtons).toHaveLength(2)
    })

    test('clicking approve button for specific request calls with correct id', () => {
      const onApprove = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T11:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} />)

      // Use data-testid to find specific button
      const approveButton = screen.getByTestId('approve-2')
      fireEvent.click(approveButton)
      expect(onApprove).toHaveBeenCalledWith('2')
    })

    test('clicking deny button for specific request calls with correct id', () => {
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T11:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onDeny={onDeny} />)

      // Use data-testid to find specific button
      const denyButton = screen.getByTestId('deny-1')
      fireEvent.click(denyButton)
      expect(onDeny).toHaveBeenCalledWith('1', 'No reason provided')
    })
  })

  describe('Task Types', () => {
    test('displays run_tests task type', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('run_tests')).toBeInTheDocument()
    })

    test('displays deploy task type', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'deploy', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('deploy')).toBeInTheDocument()
    })

    test('displays custom task types', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'custom_task_name', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('custom_task_name')).toBeInTheDocument()
    })

    test('handles task type with special characters', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'task-with_special.chars', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('task-with_special.chars')).toBeInTheDocument()
    })
  })

  describe('Timestamp Display', () => {
    test('formats timestamp correctly', () => {
      const timestamp = '2026-01-28T10:00:00Z'
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp }
      ]
      render(<ApprovalUI requests={requests} />)
      const formattedDate = new Date(timestamp).toLocaleString()
      expect(screen.getByText(formattedDate)).toBeInTheDocument()
    })

    test('displays different timestamps for multiple requests', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T15:30:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      const date1 = new Date('2026-01-28T10:00:00Z').toLocaleString()
      const date2 = new Date('2026-01-28T15:30:00Z').toLocaleString()
      expect(screen.getByText(date1)).toBeInTheDocument()
      expect(screen.getByText(date2)).toBeInTheDocument()
    })
  })

  describe('Requester Display', () => {
    test('displays requester name', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
    })

    test('displays different requesters for multiple requests', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' },
        { id: '2', requester: 'eng-suzuki', taskType: 'deploy', params: {}, timestamp: '2026-01-28T11:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })

    test('handles requester with special characters', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'user_name-123@domain', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('user_name-123@domain')).toBeInTheDocument()
    })
  })

  describe('Request Card Styling', () => {
    test('request cards have borders', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<ApprovalUI requests={requests} />)
      const card = container.querySelector('[style*="border"]')
      expect(card).toBeInTheDocument()
    })

    test('request cards have padding', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<ApprovalUI requests={requests} />)
      const card = container.querySelector('[style*="padding"]')
      expect(card).toBeInTheDocument()
    })

    test('request cards have margin', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<ApprovalUI requests={requests} />)
      const card = container.querySelector('[style*="margin"]')
      expect(card).toBeInTheDocument()
    })

    test('deny button has left margin', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      const denyButton = screen.getByRole('button', { name: /deny/i })
      expect(denyButton).toHaveStyle({ marginLeft: '10px' })
    })
  })

  describe('Edge Cases', () => {
    test('handles empty requester name', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: '', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<ApprovalUI requests={requests} />)
      expect(container).toBeInTheDocument()
    })

    test('handles empty task type', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: '', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      const { container } = render(<ApprovalUI requests={requests} />)
      expect(container).toBeInTheDocument()
    })

    test('handles params with data', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: { test: 'value', count: 123 }, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
    })

    test('handles very long task type names', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'very_long_task_type_name_that_might_break_layout', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('very_long_task_type_name_that_might_break_layout')).toBeInTheDocument()
    })

    test('handles many pending approvals', () => {
      const requests: ApprovalRequest[] = Array.from({ length: 10 }, (_, i) => ({
        id: `${i}`,
        requester: `requester-${i}`,
        taskType: `task-${i}`,
        params: {},
        timestamp: '2026-01-28T10:00:00Z'
      }))
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByText('requester-0')).toBeInTheDocument()
      expect(screen.getByText('requester-9')).toBeInTheDocument()
    })

    test('single request displays correctly', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      expect(screen.getByRole('heading', { name: /pending approvals/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /approve/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /deny/i })).toBeInTheDocument()
    })
  })

  describe('Callback Behavior', () => {
    test('onApprove is not called when not provided', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      const approveButton = screen.getByRole('button', { name: /approve/i })
      expect(() => fireEvent.click(approveButton)).not.toThrow()
    })

    test('onDeny is not called when not provided', () => {
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} />)
      const denyButton = screen.getByRole('button', { name: /deny/i })
      expect(() => fireEvent.click(denyButton)).not.toThrow()
    })

    test('only onApprove is called when approve is clicked', () => {
      const onApprove = vi.fn()
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} onDeny={onDeny} />)

      const approveButton = screen.getAllByRole('button', { name: /approve/i })[0]
      fireEvent.click(approveButton)
      expect(onApprove).toHaveBeenCalledTimes(1)
      expect(onDeny).not.toHaveBeenCalled()
    })

    test('only onDeny is called when deny is clicked', () => {
      const onApprove = vi.fn()
      const onDeny = vi.fn()
      const requests: ApprovalRequest[] = [
        { id: '1', requester: 'pm-tanaka', taskType: 'run_tests', params: {}, timestamp: '2026-01-28T10:00:00Z' }
      ]
      render(<ApprovalUI requests={requests} onApprove={onApprove} onDeny={onDeny} />)

      const denyButton = screen.getAllByRole('button', { name: /deny/i })[0]
      fireEvent.click(denyButton)
      expect(onDeny).toHaveBeenCalledTimes(1)
      expect(onApprove).not.toHaveBeenCalled()
    })
  })
})
