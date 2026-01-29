import { useState, useEffect, useCallback, useRef } from 'react';
import type { Notification } from '../types';
import { useWebSocket, WS_MESSAGE_TYPES, type WSMessage, type NotificationPayload } from './useWebSocket';

interface UseNotificationsOptions {
  useWebSocketUpdates?: boolean;
  pollingInterval?: number;
  maxNotifications?: number;
}

interface UseNotificationsReturn {
  notifications: Notification[];
  unreadCount: number;
  markRead: (id: string) => void;
  markAllRead: () => void;
  clearNotification: (id: string) => void;
  connectionStatus: 'polling' | 'websocket' | 'error';
}

export function useNotifications(options: UseNotificationsOptions = {}): UseNotificationsReturn {
  const { useWebSocketUpdates = true, pollingInterval = 10000, maxNotifications = 50 } = options;

  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [readIds, setReadIds] = useState<Set<string>>(new Set());
  const [connectionStatus, setConnectionStatus] = useState<'polling' | 'websocket' | 'error'>('polling');

  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const ws = useWebSocket({
    autoConnect: useWebSocketUpdates,
    reconnect: true,
    maxReconnectAttempts: 5,
  });

  const fetchNotifications = useCallback(async () => {
    try {
      // TODO: Replace with actual API call when backend endpoint is available
      // const data = await apiClient.getNotifications();
      // For now, keep empty array as API is not yet implemented
      setNotifications([]);
    } catch (err) {
      console.error('Failed to fetch notifications:', err);
      setConnectionStatus('error');
    }
  }, []);

  // Handle WebSocket notification updates
  const handleNotification = useCallback((message: WSMessage<NotificationPayload>) => {
    if (!message.payload) return;

    const { id, type, from, to, message: notifMessage, timestamp } = message.payload;

    const newNotification: Notification = {
      id,
      type,
      from,
      to,
      message: notifMessage,
      timestamp,
    };

    setNotifications(prevNotifications => {
      // Check if notification already exists
      if (prevNotifications.some(n => n.id === id)) {
        return prevNotifications;
      }

      // Add new notification at the beginning and limit total
      const updated = [newNotification, ...prevNotifications];
      return updated.slice(0, maxNotifications);
    });
  }, [maxNotifications]);

  // Set up WebSocket message handler
  useEffect(() => {
    if (useWebSocketUpdates) {
      ws.addMessageHandler(WS_MESSAGE_TYPES.NOTIFICATION, handleNotification as (msg: WSMessage) => void);

      return () => {
        ws.removeMessageHandler(WS_MESSAGE_TYPES.NOTIFICATION, handleNotification as (msg: WSMessage) => void);
      };
    }
  }, [useWebSocketUpdates, ws, handleNotification]);

  // Update connection status based on WebSocket state
  useEffect(() => {
    if (ws.status === 'connected' && useWebSocketUpdates) {
      setConnectionStatus('websocket');
      // Reduce polling frequency when WebSocket is connected
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      // Keep a slower poll as backup
      pollingIntervalRef.current = setInterval(fetchNotifications, 60000);
    } else if (ws.status === 'disconnected' || ws.status === 'reconnecting') {
      setConnectionStatus('polling');
      // Normal polling when WebSocket is not available
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
      pollingIntervalRef.current = setInterval(fetchNotifications, pollingInterval);
    }
  }, [ws.status, useWebSocketUpdates, fetchNotifications, pollingInterval]);

  // Initial fetch and polling setup
  useEffect(() => {
    fetchNotifications();

    // Start with normal polling
    pollingIntervalRef.current = setInterval(fetchNotifications, pollingInterval);

    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [fetchNotifications, pollingInterval]);

  const unreadCount = notifications.filter(n => !readIds.has(n.id)).length;

  const markRead = useCallback((id: string) => {
    setReadIds(prev => new Set(prev).add(id));
  }, []);

  const markAllRead = useCallback(() => {
    setReadIds(new Set(notifications.map(n => n.id)));
  }, [notifications]);

  const clearNotification = useCallback((id: string) => {
    setNotifications(prev => prev.filter(n => n.id !== id));
  }, []);

  return { notifications, unreadCount, markRead, markAllRead, clearNotification, connectionStatus };
}
