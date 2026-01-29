import { useState, useEffect, useCallback, useRef } from 'react';

// WebSocket message types
export const WS_MESSAGE_TYPES = {
  AGENT_UPDATE: 'agent_update',
  AUDIT_ENTRY: 'audit_entry',
  NOTIFICATION: 'notification',
  APPROVAL_REQUEST: 'approval_request',
  PING: 'ping',
  PONG: 'pong',
  SUBSCRIBE: 'subscribe',
  UNSUBSCRIBE: 'unsubscribe',
  ERROR: 'error',
} as const;

export type WSMessageType = typeof WS_MESSAGE_TYPES[keyof typeof WS_MESSAGE_TYPES];

export interface WSMessage<T = unknown> {
  type: WSMessageType;
  payload?: T;
  timestamp: string;
  id?: string;
}

export interface AgentUpdatePayload {
  agent_id: string;
  status: string;
  role?: string;
  endpoint?: string;
}

export interface AuditEntryPayload {
  id: string;
  from: string;
  to: string;
  event_type: string;
  summary: string;
  metadata?: Record<string, string>;
}

export interface NotificationPayload {
  id: string;
  type: string;
  from: string;
  to: string;
  message: string;
  timestamp: string;
}

export interface ApprovalRequestPayload {
  id: string;
  requester: string;
  task_type: string;
  description: string;
  params?: Record<string, unknown>;
  status: string;
}

export interface SubscribePayload {
  topics: string[];
}

export type MessageHandler<T = unknown> = (message: WSMessage<T>) => void;

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'reconnecting';

interface UseWebSocketOptions {
  url?: string;
  agentId?: string;
  autoConnect?: boolean;
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
}

interface UseWebSocketReturn {
  status: ConnectionStatus;
  lastMessage: WSMessage | null;
  sendMessage: (message: WSMessage) => void;
  subscribe: (topics: string[]) => void;
  unsubscribe: (topics: string[]) => void;
  connect: () => void;
  disconnect: () => void;
  addMessageHandler: (type: WSMessageType, handler: MessageHandler) => void;
  removeMessageHandler: (type: WSMessageType, handler: MessageHandler) => void;
}

const DEFAULT_WS_URL = import.meta.env.VITE_WS_URL || 
  `ws://${window.location.hostname}:8080/api/v1/ws`;

export function useWebSocket(options: UseWebSocketOptions = {}): UseWebSocketReturn {
  const {
    url = DEFAULT_WS_URL,
    agentId,
    autoConnect = true,
    reconnect = true,
    reconnectInterval = 1000,
    maxReconnectAttempts = 10,
    onOpen,
    onClose,
    onError,
  } = options;

  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageHandlersRef = useRef<Map<WSMessageType, Set<MessageHandler>>>(new Map());

  // Calculate reconnect delay with exponential backoff
  const getReconnectDelay = useCallback(() => {
    const attempts = reconnectAttemptsRef.current;
    const delay = reconnectInterval * Math.pow(2, Math.min(attempts, 5));
    return Math.min(delay, 30000); // Cap at 30 seconds
  }, [reconnectInterval]);

  // Send a message through WebSocket
  const sendMessage = useCallback((message: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    }
  }, []);

  // Subscribe to topics
  const subscribe = useCallback((topics: string[]) => {
    const message: WSMessage<SubscribePayload> = {
      type: WS_MESSAGE_TYPES.SUBSCRIBE,
      payload: { topics },
      timestamp: new Date().toISOString(),
    };
    sendMessage(message);
  }, [sendMessage]);

  // Unsubscribe from topics
  const unsubscribe = useCallback((topics: string[]) => {
    const message: WSMessage<SubscribePayload> = {
      type: WS_MESSAGE_TYPES.UNSUBSCRIBE,
      payload: { topics },
      timestamp: new Date().toISOString(),
    };
    sendMessage(message);
  }, [sendMessage]);

  // Add a message handler for a specific message type
  const addMessageHandler = useCallback((type: WSMessageType, handler: MessageHandler) => {
    if (!messageHandlersRef.current.has(type)) {
      messageHandlersRef.current.set(type, new Set());
    }
    messageHandlersRef.current.get(type)!.add(handler);
  }, []);

  // Remove a message handler
  const removeMessageHandler = useCallback((type: WSMessageType, handler: MessageHandler) => {
    messageHandlersRef.current.get(type)?.delete(handler);
  }, []);

  // Handle incoming messages
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message: WSMessage = JSON.parse(event.data);
      setLastMessage(message);

      // Call registered handlers for this message type
      const handlers = messageHandlersRef.current.get(message.type);
      if (handlers) {
        handlers.forEach(handler => handler(message));
      }

      // Handle ping with automatic pong
      if (message.type === WS_MESSAGE_TYPES.PING) {
        sendMessage({
          type: WS_MESSAGE_TYPES.PONG,
          timestamp: new Date().toISOString(),
        });
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  }, [sendMessage]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    setStatus('connecting');

    // Build URL with agent ID if provided
    let wsUrl = url;
    if (agentId) {
      const separator = url.includes('?') ? '&' : '?';
      wsUrl = `${url}${separator}agent_id=${encodeURIComponent(agentId)}`;
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus('connected');
      reconnectAttemptsRef.current = 0;
      onOpen?.();
    };

    ws.onclose = () => {
      setStatus('disconnected');
      wsRef.current = null;
      onClose?.();

      // Attempt reconnection
      if (reconnect && reconnectAttemptsRef.current < maxReconnectAttempts) {
        reconnectAttemptsRef.current += 1;
        const delay = getReconnectDelay();
        setStatus('reconnecting');
        
        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, delay);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      onError?.(error);
    };

    ws.onmessage = handleMessage;
  }, [url, agentId, reconnect, maxReconnectAttempts, getReconnectDelay, handleMessage, onOpen, onClose, onError]);

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    reconnectAttemptsRef.current = maxReconnectAttempts; // Prevent reconnection

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus('disconnected');
  }, [maxReconnectAttempts]);

  // Auto-connect on mount
  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  return {
    status,
    lastMessage,
    sendMessage,
    subscribe,
    unsubscribe,
    connect,
    disconnect,
    addMessageHandler,
    removeMessageHandler,
  };
}

// Helper hook to create a WebSocket context for shared connection
export function useWebSocketMessage<T = unknown>(
  ws: UseWebSocketReturn,
  type: WSMessageType,
  handler: MessageHandler<T>
): void {
  useEffect(() => {
    ws.addMessageHandler(type, handler as MessageHandler);
    return () => {
      ws.removeMessageHandler(type, handler as MessageHandler);
    };
  }, [ws, type, handler]);
}
