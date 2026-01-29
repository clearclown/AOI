package notify

import (
	"sync"
	"testing"
	"time"
)

func TestNewNotificationManager(t *testing.T) {
	nm := NewNotificationManager()
	if nm == nil {
		t.Fatal("Expected non-nil notification manager")
	}

	if nm.subscribers == nil {
		t.Error("Expected subscribers map to be initialized")
	}

	if nm.buffer == nil {
		t.Error("Expected buffer map to be initialized")
	}
}

func TestSubscribe(t *testing.T) {
	nm := NewNotificationManager()

	ch := nm.Subscribe("agent-1")
	if ch == nil {
		t.Fatal("Expected non-nil channel")
	}

	count := nm.GetSubscriberCount("agent-1")
	if count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}
}

func TestSubscribe_Multiple(t *testing.T) {
	nm := NewNotificationManager()

	ch1 := nm.Subscribe("agent-1")
	ch2 := nm.Subscribe("agent-1")

	if ch1 == ch2 {
		t.Error("Expected different channels for different subscriptions")
	}

	count := nm.GetSubscriberCount("agent-1")
	if count != 2 {
		t.Errorf("Expected 2 subscribers, got %d", count)
	}
}

func TestUnsubscribe(t *testing.T) {
	nm := NewNotificationManager()

	ch := nm.Subscribe("agent-1")

	nm.Unsubscribe("agent-1", ch)

	count := nm.GetSubscriberCount("agent-1")
	if count != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", count)
	}
}

func TestSend_WithSubscriber(t *testing.T) {
	nm := NewNotificationManager()

	ch := nm.Subscribe("agent-1")

	notif := Notification{
		ID:        "notif-1",
		Type:      "test",
		From:      "agent-2",
		To:        "agent-1",
		Message:   "Hello",
		Timestamp: time.Now(),
	}

	err := nm.Send(notif)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case received := <-ch:
		if received.ID != notif.ID {
			t.Errorf("Expected notification ID %s, got %s", notif.ID, received.ID)
		}
		if received.Message != "Hello" {
			t.Errorf("Expected message 'Hello', got %s", received.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for notification")
	}
}

func TestSend_WithoutSubscriber(t *testing.T) {
	nm := NewNotificationManager()

	notif := Notification{
		ID:        "notif-1",
		Type:      "test",
		From:      "agent-2",
		To:        "agent-1",
		Message:   "Hello",
		Timestamp: time.Now(),
	}

	err := nm.Send(notif)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Should be buffered
	count := nm.GetBufferedCount("agent-1")
	if count != 1 {
		t.Errorf("Expected 1 buffered notification, got %d", count)
	}
}

func TestSend_BufferedDelivery(t *testing.T) {
	nm := NewNotificationManager()

	// Send notification before subscriber exists
	notif := Notification{
		ID:        "notif-1",
		Type:      "test",
		From:      "agent-2",
		To:        "agent-1",
		Message:   "Buffered",
		Timestamp: time.Now(),
	}

	nm.Send(notif)

	// Now subscribe
	ch := nm.Subscribe("agent-1")

	// Should receive buffered notification
	select {
	case received := <-ch:
		if received.ID != notif.ID {
			t.Errorf("Expected notification ID %s, got %s", notif.ID, received.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for buffered notification")
	}
}

func TestBroadcast(t *testing.T) {
	nm := NewNotificationManager()

	ch1 := nm.Subscribe("agent-1")
	ch2 := nm.Subscribe("agent-2")

	notif := Notification{
		ID:        "broadcast-1",
		Type:      "announcement",
		From:      "system",
		Message:   "Broadcast message",
		Timestamp: time.Now(),
	}

	err := nm.Broadcast(notif)
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	// Both should receive
	received := 0
	timeout := time.After(200 * time.Millisecond)

	for i := 0; i < 2; i++ {
		select {
		case <-ch1:
			received++
		case <-ch2:
			received++
		case <-timeout:
			break
		}
	}

	if received != 2 {
		t.Errorf("Expected 2 notifications received, got %d", received)
	}
}

func TestClearBuffer(t *testing.T) {
	nm := NewNotificationManager()

	notif := Notification{
		ID:        "notif-1",
		Type:      "test",
		From:      "agent-2",
		To:        "agent-1",
		Message:   "Buffered",
		Timestamp: time.Now(),
	}

	nm.Send(notif)

	count := nm.GetBufferedCount("agent-1")
	if count != 1 {
		t.Errorf("Expected 1 buffered notification, got %d", count)
	}

	nm.ClearBuffer("agent-1")

	count = nm.GetBufferedCount("agent-1")
	if count != 0 {
		t.Errorf("Expected 0 buffered notifications after clear, got %d", count)
	}
}

func TestConcurrentSend(t *testing.T) {
	nm := NewNotificationManager()

	ch := nm.Subscribe("agent-1")

	numSends := 100
	var wg sync.WaitGroup
	wg.Add(numSends)

	for i := 0; i < numSends; i++ {
		go func(id int) {
			defer wg.Done()
			notif := Notification{
				ID:        string(rune('0' + (id % 10))),
				Type:      "test",
				From:      "sender",
				To:        "agent-1",
				Message:   "Concurrent",
				Timestamp: time.Now(),
			}
			nm.Send(notif)
		}(i)
	}

	wg.Wait()

	// Drain the channel
	received := 0
	timeout := time.After(500 * time.Millisecond)
drainLoop:
	for {
		select {
		case <-ch:
			received++
		case <-timeout:
			break drainLoop
		default:
			if received >= numSends {
				break drainLoop
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if received == 0 {
		t.Error("Expected to receive some notifications")
	}
}

func TestMaxBuffer(t *testing.T) {
	nm := NewNotificationManager()
	nm.maxBuffer = 5 // Set small buffer for testing

	// Send more than max buffer
	for i := 0; i < 10; i++ {
		notif := Notification{
			ID:        string(rune('0' + i)),
			Type:      "test",
			From:      "sender",
			To:        "agent-1",
			Message:   "Overflow test",
			Timestamp: time.Now(),
		}
		nm.Send(notif)
	}

	count := nm.GetBufferedCount("agent-1")
	if count > 5 {
		t.Errorf("Expected max 5 buffered notifications, got %d", count)
	}
}

func TestNotification_Fields(t *testing.T) {
	notif := Notification{
		ID:        "test-id",
		Type:      "status",
		From:      "agent-1",
		To:        "agent-2",
		Message:   "Test message",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	if notif.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", notif.ID)
	}

	if notif.Type != "status" {
		t.Errorf("Expected Type 'status', got %s", notif.Type)
	}

	if notif.Data["key"] != "value" {
		t.Error("Expected Data to contain 'key': 'value'")
	}
}
