import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAgents } from '../useAgents'
import * as apiModule from '../../services/api'

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState: number = MockWebSocket.CONNECTING
  url: string
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null

  constructor(url: string) {
    this.url = url
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    }, 10)
  }

  send = vi.fn()

  close() {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close'))
    }
  }
}

vi.mock('../../services/api', () => ({
  apiClient: {
    discoverAgents: vi.fn(),
  }
}))

describe('useAgents hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  test('fetches agents on mount', async () => {
    const mockAgents = [
      {
        id: 'agent-1',
        role: 'engineer',
        owner: 'user1',
        status: 'online' as const,
        capabilities: ['code'],
        lastSeen: '2026-01-28T10:00:00Z',
        endpoint: 'http://localhost:8080'
      }
    ]

    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue(mockAgents)

    const { result } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    expect(result.current.loading).toBe(true)
    expect(result.current.agents).toEqual([])

    // Allow initial fetch to complete
    await act(async () => {
      // Flush microtasks to let the fetch resolve
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.agents).toEqual(mockAgents)
    expect(result.current.error).toBeNull()
  })

  test('handles fetch error', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockRejectedValue(
      new Error('Network error')
    )

    const { result } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBe('Network error')
    expect(result.current.agents).toEqual([])
  })

  test('refresh function triggers new fetch', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue([])

    const { result } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(apiModule.apiClient.discoverAgents).toHaveBeenCalledTimes(1)

    await act(async () => {
      await result.current.refresh()
    })

    expect(apiModule.apiClient.discoverAgents).toHaveBeenCalledTimes(2)
  })

  test('returns agents array', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue([])

    const { result } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(Array.isArray(result.current.agents)).toBe(true)
  })

  test('provides error state', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue([])

    const { result } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  test('provides connection status', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue([])

    const { result, unmount } = renderHook(() => useAgents({ useWebSocketUpdates: false }))

    await act(async () => {
      await Promise.resolve()
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.connectionStatus).toBe('polling')
    
    unmount()
  })

  test('uses polling interval option', async () => {
    vi.spyOn(apiModule.apiClient, 'discoverAgents').mockResolvedValue([])

    const { unmount } = renderHook(() => useAgents({ 
      useWebSocketUpdates: false,
      pollingInterval: 5000 
    }))

    // Initial fetch
    await act(async () => {
      await Promise.resolve()
    })

    const initialCalls = (apiModule.apiClient.discoverAgents as ReturnType<typeof vi.fn>).mock.calls.length
    expect(initialCalls).toBeGreaterThanOrEqual(1)

    // Advance to trigger one more poll
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000)
    })

    const afterPollCalls = (apiModule.apiClient.discoverAgents as ReturnType<typeof vi.fn>).mock.calls.length
    expect(afterPollCalls).toBeGreaterThan(initialCalls)

    unmount()
  })
})
