import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient } from '../services/api';
import type { Agent } from '../types';
import { useWebSocket, WS_MESSAGE_TYPES, type WSMessage, type AgentUpdatePayload } from './useWebSocket';

interface UseAgentsOptions {
  useWebSocketUpdates?: boolean;
  pollingInterval?: number;
}

interface UseAgentsReturn {
  agents: Agent[];
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  connectionStatus: 'polling' | 'websocket' | 'error';
}

export function useAgents(options: UseAgentsOptions = {}): UseAgentsReturn {
  const { useWebSocketUpdates = true, pollingInterval = 10000 } = options;

  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'polling' | 'websocket' | 'error'>('polling');

  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const ws = useWebSocket({
    autoConnect: useWebSocketUpdates,
    reconnect: true,
    maxReconnectAttempts: 5,
  });

  const fetchAgents = useCallback(async () => {
    try {
      setError(null);
      const data = await apiClient.discoverAgents();
      setAgents(data);
      setLoading(false);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch agents';
      setError(errorMessage);
      setLoading(false);
      setConnectionStatus('error');
    }
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchAgents();
  }, [fetchAgents]);

  // Handle WebSocket agent updates
  const handleAgentUpdate = useCallback((message: WSMessage<AgentUpdatePayload>) => {
    if (!message.payload) return;

    const { agent_id, status, role, endpoint } = message.payload;

    setAgents(prevAgents => {
      const existingIndex = prevAgents.findIndex(a => a.id === agent_id);

      if (existingIndex >= 0) {
        // Update existing agent
        const updated = [...prevAgents];
        updated[existingIndex] = {
          ...updated[existingIndex],
          status: status as Agent['status'],
          ...(role && { role }),
          ...(endpoint && { endpoint }),
          lastSeen: new Date().toISOString(),
        };
        return updated;
      } else if (role && status) {
        // Add new agent (need minimal info)
        return [...prevAgents, {
          id: agent_id,
          role: role as Agent['role'],
          owner: '',
          status: status as Agent['status'],
          capabilities: [],
          lastSeen: new Date().toISOString(),
          endpoint: endpoint || '',
        }];
      }

      return prevAgents;
    });
  }, []);

  // Set up WebSocket message handler
  useEffect(() => {
    if (useWebSocketUpdates) {
      ws.addMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handleAgentUpdate as (msg: WSMessage) => void);

      return () => {
        ws.removeMessageHandler(WS_MESSAGE_TYPES.AGENT_UPDATE, handleAgentUpdate as (msg: WSMessage) => void);
      };
    }
  }, [useWebSocketUpdates, ws, handleAgentUpdate]);

  // Update connection status based on WebSocket state
  useEffect(() => {
    if (ws.status === 'connected' && useWebSocketUpdates) {
      setConnectionStatus('websocket');
      // Reduce polling frequency when WebSocket is connected
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      // Keep a slower poll as backup for full refreshes
      pollingIntervalRef.current = setInterval(fetchAgents, 30000);
    } else if (ws.status === 'disconnected' || ws.status === 'reconnecting') {
      setConnectionStatus('polling');
      // Increase polling frequency when WebSocket is not available
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      pollingIntervalRef.current = setInterval(fetchAgents, pollingInterval);
    }
  }, [ws.status, useWebSocketUpdates, fetchAgents, pollingInterval]);

  // Initial fetch and polling setup
  useEffect(() => {
    fetchAgents();

    // Start with normal polling
    pollingIntervalRef.current = setInterval(fetchAgents, pollingInterval);

    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [fetchAgents, pollingInterval]);

  return { agents, loading, error, refresh, connectionStatus };
}
