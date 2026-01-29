import { useState, useEffect, useCallback, useRef } from 'react';
import type { AuditEntry } from '../types';
import { useWebSocket, WS_MESSAGE_TYPES, type WSMessage, type AuditEntryPayload } from './useWebSocket';

interface UseAuditLogOptions {
  useWebSocketUpdates?: boolean;
  pollingInterval?: number;
  maxEntries?: number;
}

interface UseAuditLogReturn {
  entries: AuditEntry[];
  loading: boolean;
  error: string | null;
  connectionStatus: 'polling' | 'websocket' | 'error';
}

export function useAuditLog(options: UseAuditLogOptions = {}): UseAuditLogReturn {
  const { useWebSocketUpdates = true, pollingInterval = 5000, maxEntries = 100 } = options;

  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'polling' | 'websocket' | 'error'>('polling');

  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const ws = useWebSocket({
    autoConnect: useWebSocketUpdates,
    reconnect: true,
    maxReconnectAttempts: 5,
  });

  const fetchAuditLog = useCallback(async () => {
    try {
      setError(null);
      // TODO: Replace with actual API call when backend endpoint is available
      // const data = await apiClient.getAuditLog();
      // For now, keep empty array as API is not yet implemented
      setEntries([]);
      setLoading(false);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch audit log';
      setError(errorMessage);
      setLoading(false);
      setConnectionStatus('error');
    }
  }, []);

  // Handle WebSocket audit entry updates
  const handleAuditEntry = useCallback((message: WSMessage<AuditEntryPayload>) => {
    if (!message.payload) return;

    const { id, from, to, event_type, summary, metadata } = message.payload;

    const newEntry: AuditEntry = {
      id,
      from,
      to,
      eventType: event_type as AuditEntry['eventType'],
      summary,
      timestamp: message.timestamp,
      metadata,
    };

    setEntries(prevEntries => {
      // Check if entry already exists
      if (prevEntries.some(e => e.id === id)) {
        return prevEntries;
      }

      // Add new entry at the beginning and limit total entries
      const updated = [newEntry, ...prevEntries];
      return updated.slice(0, maxEntries);
    });
  }, [maxEntries]);

  // Set up WebSocket message handler
  useEffect(() => {
    if (useWebSocketUpdates) {
      ws.addMessageHandler(WS_MESSAGE_TYPES.AUDIT_ENTRY, handleAuditEntry as (msg: WSMessage) => void);

      return () => {
        ws.removeMessageHandler(WS_MESSAGE_TYPES.AUDIT_ENTRY, handleAuditEntry as (msg: WSMessage) => void);
      };
    }
  }, [useWebSocketUpdates, ws, handleAuditEntry]);

  // Update connection status based on WebSocket state
  useEffect(() => {
    if (ws.status === 'connected' && useWebSocketUpdates) {
      setConnectionStatus('websocket');
      // Reduce polling frequency when WebSocket is connected
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      // Keep a slower poll as backup
      pollingIntervalRef.current = setInterval(fetchAuditLog, 30000);
    } else if (ws.status === 'disconnected' || ws.status === 'reconnecting') {
      setConnectionStatus('polling');
      // Normal polling when WebSocket is not available
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      pollingIntervalRef.current = setInterval(fetchAuditLog, pollingInterval);
    }
  }, [ws.status, useWebSocketUpdates, fetchAuditLog, pollingInterval]);

  // Initial fetch and polling setup
  useEffect(() => {
    fetchAuditLog();

    // Start with normal polling
    pollingIntervalRef.current = setInterval(fetchAuditLog, pollingInterval);

    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [fetchAuditLog, pollingInterval]);

  return { entries, loading, error, connectionStatus };
}
