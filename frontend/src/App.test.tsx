import { render, screen, fireEvent } from '@testing-library/react'
import App from './App'

describe('App Component', () => {
  describe('Initial Render', () => {
    test('renders app title', () => {
      render(<App />)
      expect(screen.getByText(/AOI Protocol/i)).toBeInTheDocument()
    })

    test('renders main heading', () => {
      render(<App />)
      expect(screen.getByRole('heading', { name: /AOI Protocol Dashboard/i })).toBeInTheDocument()
    })

    test('renders navigation tabs', () => {
      render(<App />)
      expect(screen.getByRole('button', { name: /dashboard/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /audit/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /approvals/i })).toBeInTheDocument()
    })

    test('starts with dashboard tab active', () => {
      render(<App />)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
    })

    test('renders navigation in correct order', () => {
      render(<App />)
      const buttons = screen.getAllByRole('button')
      expect(buttons[0]).toHaveTextContent(/dashboard/i)
      expect(buttons[1]).toHaveTextContent(/audit/i)
      expect(buttons[2]).toHaveTextContent(/approvals/i)
    })
  })

  describe('Tab Navigation', () => {
    test('clicking audit tab shows audit log', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })

    test('clicking approvals tab shows approvals', () => {
      render(<App />)
      const approvalsButton = screen.getByRole('button', { name: /approvals/i })
      fireEvent.click(approvalsButton)
      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })

    test('clicking dashboard tab shows dashboard', () => {
      render(<App />)
      const dashboardButton = screen.getByRole('button', { name: /dashboard/i })
      fireEvent.click(dashboardButton)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
    })

    test('navigation between tabs works correctly', () => {
      render(<App />)

      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()

      const approvalsButton = screen.getByRole('button', { name: /approvals/i })
      fireEvent.click(approvalsButton)
      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()

      const dashboardButton = screen.getByRole('button', { name: /dashboard/i })
      fireEvent.click(dashboardButton)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
    })

    test('switching tabs hides previous tab content', () => {
      render(<App />)

      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()

      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)

      expect(screen.queryByRole('heading', { name: /agent dashboard/i })).not.toBeInTheDocument()
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })

    test('can switch back and forth between tabs', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      const dashboardButton = screen.getByRole('button', { name: /dashboard/i })

      fireEvent.click(auditButton)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()

      fireEvent.click(dashboardButton)
      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()

      fireEvent.click(auditButton)
      expect(screen.getByRole('heading', { name: /audit log/i })).toBeInTheDocument()
    })

    test('clicking same tab multiple times maintains view', () => {
      render(<App />)
      const dashboardButton = screen.getByRole('button', { name: /dashboard/i })

      fireEvent.click(dashboardButton)
      fireEvent.click(dashboardButton)
      fireEvent.click(dashboardButton)

      expect(screen.getByRole('heading', { name: /agent dashboard/i })).toBeInTheDocument()
    })
  })

  describe('Dashboard Tab Content', () => {
    test('dashboard shows agent data', () => {
      render(<App />)
      expect(screen.getByText(/eng-local/i)).toBeInTheDocument()
    })

    test('dashboard shows agent role', () => {
      render(<App />)
      expect(screen.getByText(/engineer/i)).toBeInTheDocument()
    })

    test('dashboard shows agent status', () => {
      render(<App />)
      expect(screen.getByText(/online/i)).toBeInTheDocument()
    })

    test('dashboard displays last seen information', () => {
      render(<App />)
      expect(screen.getByText(/last seen:/i)).toBeInTheDocument()
    })
  })

  describe('Audit Log Tab Content', () => {
    test('audit log shows entry when switched to', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)

      expect(screen.getByText(/pm-tanaka/i)).toBeInTheDocument()
      expect(screen.getByText(/eng-suzuki/i)).toBeInTheDocument()
    })

    test('audit log shows event summary', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)

      expect(screen.getByText(/status check/i)).toBeInTheDocument()
    })

    test('audit log shows event type', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)

      // Event type appears in both filter dropdown and entry display
      const queryElements = screen.getAllByText('query')
      expect(queryElements.length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('Approvals Tab Content', () => {
    test('approvals shows empty state with no requests', () => {
      render(<App />)
      const approvalsButton = screen.getByRole('button', { name: /approvals/i })
      fireEvent.click(approvalsButton)

      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })

    test('approvals tab is accessible', () => {
      render(<App />)
      const approvalsButton = screen.getByRole('button', { name: /approvals/i })
      fireEvent.click(approvalsButton)

      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })
  })

  describe('Layout and Styling', () => {
    test('has padding on main container', () => {
      const { container } = render(<App />)
      const mainDiv = container.firstChild as HTMLElement
      expect(mainDiv).toHaveStyle({ padding: '20px' })
    })

    test('uses system font family', () => {
      const { container } = render(<App />)
      const mainDiv = container.firstChild as HTMLElement
      expect(mainDiv).toHaveStyle({ fontFamily: 'system-ui' })
    })

    test('navigation has margin bottom', () => {
      const { container } = render(<App />)
      const nav = container.querySelector('nav')
      expect(nav).toHaveStyle({ marginBottom: '20px' })
    })

    test('navigation buttons have spacing', () => {
      const { container } = render(<App />)
      const buttons = container.querySelectorAll('nav button')
      expect(buttons[0]).toHaveStyle({ marginRight: '10px' })
      expect(buttons[1]).toHaveStyle({ marginRight: '10px' })
    })
  })

  describe('Component Integration', () => {
    test('dashboard component receives agent data', () => {
      render(<App />)
      expect(screen.getByText('eng-local')).toBeInTheDocument()
    })

    test('audit log component receives entry data', () => {
      render(<App />)
      const auditButton = screen.getByRole('button', { name: /audit/i })
      fireEvent.click(auditButton)

      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
    })

    test('approval component receives empty array', () => {
      render(<App />)
      const approvalsButton = screen.getByRole('button', { name: /approvals/i })
      fireEvent.click(approvalsButton)

      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })
  })

  describe('Mock Data', () => {
    test('has mock agent with correct id', () => {
      render(<App />)
      expect(screen.getByText('eng-local')).toBeInTheDocument()
    })

    test('has mock agent with engineer role', () => {
      render(<App />)
      expect(screen.getByText('engineer')).toBeInTheDocument()
    })

    test('has mock agent with online status', () => {
      render(<App />)
      expect(screen.getByText('online')).toBeInTheDocument()
    })

    test('has mock audit entry from pm-tanaka', () => {
      render(<App />)
      fireEvent.click(screen.getByRole('button', { name: /audit/i }))
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
    })

    test('has mock audit entry to eng-suzuki', () => {
      render(<App />)
      fireEvent.click(screen.getByRole('button', { name: /audit/i }))
      expect(screen.getByText('eng-suzuki')).toBeInTheDocument()
    })

    test('has no approval requests by default', () => {
      render(<App />)
      fireEvent.click(screen.getByRole('button', { name: /approvals/i }))
      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    test('all navigation buttons are accessible', () => {
      render(<App />)
      const dashboardBtn = screen.getByRole('button', { name: /dashboard/i })
      const auditBtn = screen.getByRole('button', { name: /audit/i })
      const approvalsBtn = screen.getByRole('button', { name: /approvals/i })

      expect(dashboardBtn).toBeInTheDocument()
      expect(auditBtn).toBeInTheDocument()
      expect(approvalsBtn).toBeInTheDocument()
    })

    test('heading is properly structured', () => {
      render(<App />)
      const mainHeading = screen.getByRole('heading', { level: 1 })
      expect(mainHeading).toHaveTextContent(/AOI Protocol Dashboard/i)
    })

    test('navigation is semantic', () => {
      const { container } = render(<App />)
      const nav = container.querySelector('nav')
      expect(nav).toBeInTheDocument()
    })
  })

  describe('Tab State Persistence', () => {
    test('dashboard tab retains state when switching back', () => {
      render(<App />)
      expect(screen.getByText('eng-local')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: /audit/i }))
      fireEvent.click(screen.getByRole('button', { name: /dashboard/i }))

      expect(screen.getByText('eng-local')).toBeInTheDocument()
    })

    test('audit tab retains state when switching back', () => {
      render(<App />)
      fireEvent.click(screen.getByRole('button', { name: /audit/i }))
      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: /dashboard/i }))
      fireEvent.click(screen.getByRole('button', { name: /audit/i }))

      expect(screen.getByText('pm-tanaka')).toBeInTheDocument()
    })

    test('approvals tab retains state when switching back', () => {
      render(<App />)
      fireEvent.click(screen.getByRole('button', { name: /approvals/i }))
      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: /dashboard/i }))
      fireEvent.click(screen.getByRole('button', { name: /approvals/i }))

      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })
  })

  describe('Edge Cases', () => {
    test('renders without errors', () => {
      expect(() => render(<App />)).not.toThrow()
    })

    test('handles rapid tab switching', () => {
      render(<App />)
      const dashboardBtn = screen.getByRole('button', { name: /dashboard/i })
      const auditBtn = screen.getByRole('button', { name: /audit/i })
      const approvalsBtn = screen.getByRole('button', { name: /approvals/i })

      fireEvent.click(auditBtn)
      fireEvent.click(approvalsBtn)
      fireEvent.click(dashboardBtn)
      fireEvent.click(auditBtn)
      fireEvent.click(approvalsBtn)

      expect(screen.getByText(/no pending approvals/i)).toBeInTheDocument()
    })

    test('all navigation buttons are clickable', () => {
      render(<App />)
      const buttons = screen.getAllByRole('button')

      buttons.forEach(button => {
        expect(() => fireEvent.click(button)).not.toThrow()
      })
    })
  })
})
