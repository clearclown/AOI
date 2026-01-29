package notify

import (
	"sync"
	"time"
)

// Notification represents a message sent between agents
type Notification struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NotificationManager manages notification subscriptions and delivery
type NotificationManager struct {
	subscribers map[string][]chan Notification
	buffer      map[string][]Notification // Buffer for offline agents
	mu          sync.RWMutex
	maxBuffer   int
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		subscribers: make(map[string][]chan Notification),
		buffer:      make(map[string][]Notification),
		maxBuffer:   100, // Maximum buffered notifications per agent
	}
}

// Subscribe registers a channel to receive notifications for an agent
func (nm *NotificationManager) Subscribe(agentID string) chan Notification {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	ch := make(chan Notification, 10)
	nm.subscribers[agentID] = append(nm.subscribers[agentID], ch)

	// Deliver any buffered notifications
	if buffered, ok := nm.buffer[agentID]; ok {
		go func() {
			for _, notif := range buffered {
				select {
				case ch <- notif:
				default:
					// Channel full, skip
				}
			}
		}()
		delete(nm.buffer, agentID)
	}

	return ch
}

// Unsubscribe removes a subscription channel
func (nm *NotificationManager) Unsubscribe(agentID string, ch chan Notification) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if subs, ok := nm.subscribers[agentID]; ok {
		for i, sub := range subs {
			if sub == ch {
				// Remove this subscription
				nm.subscribers[agentID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}

		// Clean up if no more subscribers
		if len(nm.subscribers[agentID]) == 0 {
			delete(nm.subscribers, agentID)
		}
	}
}

// Send sends a notification to a specific agent
func (nm *NotificationManager) Send(notif Notification) error {
	nm.mu.RLock()
	subs, hasSubscribers := nm.subscribers[notif.To]
	nm.mu.RUnlock()

	if !hasSubscribers || len(subs) == 0 {
		// No active subscribers, buffer the notification
		nm.bufferNotification(notif)
		return nil
	}

	// Send to all subscribers
	for _, ch := range subs {
		select {
		case ch <- notif:
		default:
			// Channel full, skip this subscriber
		}
	}

	return nil
}

// Broadcast sends a notification to all subscribed agents
func (nm *NotificationManager) Broadcast(notif Notification) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for agentID, subs := range nm.subscribers {
		// Create a copy with the correct recipient
		n := notif
		n.To = agentID

		for _, ch := range subs {
			select {
			case ch <- n:
			default:
				// Channel full, skip this subscriber
			}
		}
	}

	return nil
}

// bufferNotification stores a notification for an offline agent
func (nm *NotificationManager) bufferNotification(notif Notification) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, ok := nm.buffer[notif.To]; !ok {
		nm.buffer[notif.To] = make([]Notification, 0)
	}

	// Add to buffer if not full
	if len(nm.buffer[notif.To]) < nm.maxBuffer {
		nm.buffer[notif.To] = append(nm.buffer[notif.To], notif)
	}
}

// GetBufferedCount returns the number of buffered notifications for an agent
func (nm *NotificationManager) GetBufferedCount(agentID string) int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if buffered, ok := nm.buffer[agentID]; ok {
		return len(buffered)
	}
	return 0
}

// GetSubscriberCount returns the number of active subscribers for an agent
func (nm *NotificationManager) GetSubscriberCount(agentID string) int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if subs, ok := nm.subscribers[agentID]; ok {
		return len(subs)
	}
	return 0
}

// ClearBuffer clears all buffered notifications for an agent
func (nm *NotificationManager) ClearBuffer(agentID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.buffer, agentID)
}
