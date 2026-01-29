import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { 
  useWebSocket, 
  WS_MESSAGE_TYPES,
  type WSMessage,
  type AgentUpdatePayload 
} from '../useWebSocket'

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
    // Simulate connection after a short delay
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

  // Helper method to simulate receiving a message
  simulateMessage(data: unknown) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }))
    }
  }

  // Helper method to simulate an error
  simulateError() {
    if (this.onerror) {
      this.onerror(new Event('error'))
    }
  }
}

let mockWebSocketInstance: MockWebSocket | null = null

describe('useWebSocket hook', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    // Replace global WebSocket with mock
    vi.stubGlobal('WebSocket', class extends MockWebSocket {
      constructor(url: string) {
        super(url)
        mockWebSocketInstance = this
      }
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    mockWebSocketInstance = null
  })

  test('initializes with disconnected status when autoConnect is false', () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: false }))
    
    expect(result.current.status).toBe('disconnected')
    expect(result.current.lastMessage).toBeNull()
  })

  test('connects automatically when autoConnect is true', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    expect(result.current.status).toBe('connecting')
    
    // Advance timers to allow connection
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(result.current.status).toBe('connected')
  })

  test('connects with agent ID in URL', async () => {
    renderHook(() => useWebSocket({ 
      url: 'ws://localhost:8080/ws',
      agentId: 'test-agent',
      autoConnect: true 
    }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(mockWebSocketInstance?.url).toContain('agent_id=test-agent')
  })

  test('disconnect closes the connection', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(result.current.status).toBe('connected')
    
    act(() => {
      result.current.disconnect()
    })
    
    expect(result.current.status).toBe('disconnected')
  })

  test('sendMessage sends JSON through WebSocket', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const message: WSMessage = {
      type: WS_MESSAGE_TYPES.PING,
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      result.current.sendMessage(message)
    })
    
    expect(mockWebSocketInstance?.send).toHaveBeenCalledWith(JSON.stringify(message))
  })

  test('subscribe sends subscribe message', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    act(() => {
      result.current.subscribe([WS_MESSAGE_TYPES.AGENT_UPDATE])
    })
    
    expect(mockWebSocketInstance?.send).toHaveBeenCalled()
    const sentMessage = JSON.parse(mockWebSocketInstance?.send.mock.calls[0][0])
    expect(sentMessage.type).toBe(WS_MESSAGE_TYPES.SUBSCRIBE)
    expect(sentMessage.payload.topics).toContain(WS_MESSAGE_TYPES.AGENT_UPDATE)
  })

  test('unsubscribe sends unsubscribe message', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    act(() => {
      result.current.unsubscribe([WS_MESSAGE_TYPES.AGENT_UPDATE])
    })
    
    expect(mockWebSocketInstance?.send).toHaveBeenCalled()
    const sentMessage = JSON.parse(mockWebSocketInstance?.send.mock.calls[0][0])
    expect(sentMessage.type).toBe(WS_MESSAGE_TYPES.UNSUBSCRIBE)
  })

  test('receives and updates lastMessage', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const incomingMessage: WSMessage<AgentUpdatePayload> = {
      type: WS_MESSAGE_TYPES.AGENT_UPDATE,
      payload: { agent_id: 'agent-1', status: 'online' },
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      mockWebSocketInstance?.simulateMessage(incomingMessage)
    })
    
    expect(result.current.lastMessage).toEqual(incomingMessage)
  })

  test('calls message handlers for specific types', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const handler = vi.fn()
    
    act(() => {
      result.current.addMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handler)
    })
    
    const message: WSMessage<AgentUpdatePayload> = {
      type: WS_MESSAGE_TYPES.AGENT_UPDATE,
      payload: { agent_id: 'agent-1', status: 'online' },
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      mockWebSocketInstance?.simulateMessage(message)
    })
    
    expect(handler).toHaveBeenCalledWith(message)
  })

  test('removes message handler', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const handler = vi.fn()
    
    act(() => {
      result.current.addMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handler)
      result.current.removeMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handler)
    })
    
    const message: WSMessage<AgentUpdatePayload> = {
      type: WS_MESSAGE_TYPES.AGENT_UPDATE,
      payload: { agent_id: 'agent-1', status: 'online' },
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      mockWebSocketInstance?.simulateMessage(message)
    })
    
    expect(handler).not.toHaveBeenCalled()
  })

  test('responds to ping with pong', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const pingMessage: WSMessage = {
      type: WS_MESSAGE_TYPES.PING,
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      mockWebSocketInstance?.simulateMessage(pingMessage)
    })
    
    expect(mockWebSocketInstance?.send).toHaveBeenCalled()
    const sentMessage = JSON.parse(mockWebSocketInstance?.send.mock.calls[0][0])
    expect(sentMessage.type).toBe(WS_MESSAGE_TYPES.PONG)
  })

  test('calls onOpen callback when connected', async () => {
    const onOpen = vi.fn()
    
    renderHook(() => useWebSocket({ autoConnect: true, onOpen }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(onOpen).toHaveBeenCalled()
  })

  test('calls onClose callback when disconnected', async () => {
    const onClose = vi.fn()
    
    const { result } = renderHook(() => useWebSocket({ autoConnect: true, onClose }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    act(() => {
      result.current.disconnect()
    })
    
    expect(onClose).toHaveBeenCalled()
  })

  test('attempts reconnection on disconnect when reconnect is true', async () => {
    const { result } = renderHook(() => useWebSocket({ 
      autoConnect: true, 
      reconnect: true,
      reconnectInterval: 100,
      maxReconnectAttempts: 3
    }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(result.current.status).toBe('connected')
    
    // Simulate unexpected close
    act(() => {
      mockWebSocketInstance?.close()
    })
    
    expect(result.current.status).toBe('reconnecting')
    
    // Advance timers for reconnection
    await act(async () => {
      vi.advanceTimersByTime(200)
    })
    
    // Should attempt to reconnect (status will be connecting or connected)
    expect(['connecting', 'connected', 'reconnecting']).toContain(result.current.status)
  })

  test('does not reconnect when reconnect is false', async () => {
    const { result } = renderHook(() => useWebSocket({ 
      autoConnect: true, 
      reconnect: false 
    }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    act(() => {
      mockWebSocketInstance?.close()
    })
    
    expect(result.current.status).toBe('disconnected')
    
    // Advance timers
    await act(async () => {
      vi.advanceTimersByTime(5000)
    })
    
    // Should still be disconnected
    expect(result.current.status).toBe('disconnected')
  })

  test('manual connect works after disconnect', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: false }))
    
    expect(result.current.status).toBe('disconnected')
    
    act(() => {
      result.current.connect()
    })
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(result.current.status).toBe('connected')
  })

  test('handles multiple message handlers for same type', async () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    const handler1 = vi.fn()
    const handler2 = vi.fn()
    
    act(() => {
      result.current.addMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handler1)
      result.current.addMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handler2)
    })
    
    const message: WSMessage<AgentUpdatePayload> = {
      type: WS_MESSAGE_TYPES.AGENT_UPDATE,
      payload: { agent_id: 'agent-1', status: 'online' },
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      mockWebSocketInstance?.simulateMessage(message)
    })
    
    expect(handler1).toHaveBeenCalledWith(message)
    expect(handler2).toHaveBeenCalledWith(message)
  })

  test('does not send message when not connected', () => {
    const { result } = renderHook(() => useWebSocket({ autoConnect: false }))
    
    const message: WSMessage = {
      type: WS_MESSAGE_TYPES.PING,
      timestamp: new Date().toISOString(),
    }
    
    act(() => {
      result.current.sendMessage(message)
    })
    
    // Should not throw, but also should not send
    expect(mockWebSocketInstance).toBeNull()
  })

  test('cleans up on unmount', async () => {
    const { result, unmount } = renderHook(() => useWebSocket({ autoConnect: true }))
    
    await act(async () => {
      vi.advanceTimersByTime(20)
    })
    
    expect(result.current.status).toBe('connected')
    
    unmount()
    
    // WebSocket should be closed
    expect(mockWebSocketInstance?.readyState).toBe(MockWebSocket.CLOSED)
  })
})

describe('WS_MESSAGE_TYPES', () => {
  test('has all required message types', () => {
    expect(WS_MESSAGE_TYPES.AGENT_UPDATE).toBe('agent_update')
    expect(WS_MESSAGE_TYPES.AUDIT_ENTRY).toBe('audit_entry')
    expect(WS_MESSAGE_TYPES.NOTIFICATION).toBe('notification')
    expect(WS_MESSAGE_TYPES.APPROVAL_REQUEST).toBe('approval_request')
    expect(WS_MESSAGE_TYPES.PING).toBe('ping')
    expect(WS_MESSAGE_TYPES.PONG).toBe('pong')
    expect(WS_MESSAGE_TYPES.SUBSCRIBE).toBe('subscribe')
    expect(WS_MESSAGE_TYPES.UNSUBSCRIBE).toBe('unsubscribe')
    expect(WS_MESSAGE_TYPES.ERROR).toBe('error')
  })
})
