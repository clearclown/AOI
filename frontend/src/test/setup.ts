import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Mock WebSocket globally for all tests
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState: number = MockWebSocket.CLOSED
  url: string
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null

  constructor(url: string) {
    this.url = url
    this.readyState = MockWebSocket.CONNECTING
    // Simulate failed connection in tests (silence errors)
    setTimeout(() => {
      this.readyState = MockWebSocket.CLOSED
      if (this.onerror) {
        this.onerror(new Event('error'))
      }
      if (this.onclose) {
        this.onclose(new CloseEvent('close'))
      }
    }, 0)
  }

  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close'))
    }
  })
}

// Suppress WebSocket error logging in tests
const originalConsoleError = console.error
console.error = (...args: unknown[]) => {
  if (typeof args[0] === 'string' && args[0].includes('WebSocket error')) {
    return
  }
  originalConsoleError(...args)
}

vi.stubGlobal('WebSocket', MockWebSocket)

// jsdom does not implement scrollIntoView; add a no-op stub
window.HTMLElement.prototype.scrollIntoView = vi.fn()
