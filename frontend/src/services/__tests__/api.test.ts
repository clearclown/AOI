import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { apiClient } from '../api'

describe('AOI API Client', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    global.fetch = originalFetch
    vi.clearAllMocks()
  })

  describe('getHealth', () => {
    test('returns health status on success', async () => {
      const mockResponse = { status: 'ok' }
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })

      const result = await apiClient.getHealth()
      expect(result).toEqual(mockResponse)
      expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/health'))
    })

    test('throws error on HTTP error', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })

      await expect(apiClient.getHealth()).rejects.toThrow('HTTP 500: Internal Server Error')
    })

    test('throws error on network failure', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error('Network error'))

      await expect(apiClient.getHealth()).rejects.toThrow('Network error')
    })
  })

  describe('discoverAgents', () => {
    test('returns agent list on success', async () => {
      const mockAgents = [
        {
          id: 'agent-1',
          role: 'engineer',
          owner: 'user1',
          status: 'online',
          capabilities: ['code', 'review'],
          lastSeen: '2026-01-28T10:00:00Z',
          endpoint: 'http://localhost:8080'
        }
      ]

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          result: mockAgents,
          id: 1
        }),
      })

      const result = await apiClient.discoverAgents()
      expect(result).toEqual(mockAgents)
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/rpc'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: expect.stringContaining('aoi.discover')
        })
      )
    })

    test('throws error on JSON-RPC error', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          error: { code: -32603, message: 'Internal error' },
          id: 1
        }),
      })

      await expect(apiClient.discoverAgents()).rejects.toThrow('Internal error')
    })

    test('handles empty agent list', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          result: [],
          id: 1
        }),
      })

      const result = await apiClient.discoverAgents()
      expect(result).toEqual([])
    })
  })

  describe('queryAgent', () => {
    test('returns query result on success', async () => {
      const mockResult = {
        answer: 'Test answer',
        confidence: 0.95,
        sources: ['source1', 'source2']
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          result: mockResult,
          id: 1
        }),
      })

      const result = await apiClient.queryAgent('agent-1', 'test query')
      expect(result).toEqual(mockResult)
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/rpc'),
        expect.objectContaining({
          method: 'POST',
          body: expect.stringContaining('aoi.query')
        })
      )
    })

    test('includes agent ID and query in request', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          result: { answer: 'test', confidence: 1, sources: [] },
          id: 1
        }),
      })

      await apiClient.queryAgent('test-agent', 'what is this?')

      const callArgs = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0]
      const body = JSON.parse(callArgs[1].body)
      expect(body.params).toEqual({
        agent_id: 'test-agent',
        query: 'what is this?'
      })
    })

    test('throws error on invalid response', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          error: { code: -32600, message: 'Invalid Request' },
          id: 1
        }),
      })

      await expect(apiClient.queryAgent('agent-1', 'query')).rejects.toThrow('Invalid Request')
    })
  })

  describe('getStatus', () => {
    test('returns status on success', async () => {
      const mockStatus = {
        id: 'agent-1',
        role: 'engineer',
        uptime: 3600,
        queriesHandled: 42,
        connectedAgents: 5
      }

      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          jsonrpc: '2.0',
          result: mockStatus,
          id: 1
        }),
      })

      const result = await apiClient.getStatus()
      expect(result).toEqual(mockStatus)
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/rpc'),
        expect.objectContaining({
          body: expect.stringContaining('aoi.status')
        })
      )
    })

    test('throws error on server error', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
        status: 503,
        statusText: 'Service Unavailable',
      })

      await expect(apiClient.getStatus()).rejects.toThrow('HTTP 503: Service Unavailable')
    })
  })

  describe('JSON-RPC protocol', () => {
    test('increments request ID for each call', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        ok: true,
        json: async () => ({ jsonrpc: '2.0', result: [], id: 1 }),
      })

      await apiClient.discoverAgents()
      await apiClient.discoverAgents()

      const calls = (global.fetch as ReturnType<typeof vi.fn>).mock.calls
      const body1 = JSON.parse(calls[0][1].body)
      const body2 = JSON.parse(calls[1][1].body)

      expect(body2.id).toBeGreaterThan(body1.id)
    })

    test('uses correct JSON-RPC version', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ jsonrpc: '2.0', result: [], id: 1 }),
      })

      await apiClient.discoverAgents()

      const callArgs = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0]
      const body = JSON.parse(callArgs[1].body)
      expect(body.jsonrpc).toBe('2.0')
    })

    test('includes method name in request', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ jsonrpc: '2.0', result: {}, id: 1 }),
      })

      await apiClient.getStatus()

      const callArgs = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0]
      const body = JSON.parse(callArgs[1].body)
      expect(body.method).toBe('aoi.status')
    })
  })
})
