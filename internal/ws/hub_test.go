package ws

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestClient(userID string) *Client {
	return &Client{
		conn:   nil, // Not needed for hub tests
		userID: userID,
		send:   make(chan Message, 256),
		logger: testLogger(),
	}
}

// TestNewHub verifies that NewHub creates a hub with no clients.
func TestNewHub(t *testing.T) {
	hub := NewHub(testLogger())

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("hub.clients map is nil")
	}

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", hub.ClientCount())
	}
}

// TestRegister verifies that Register adds a client and increments ClientCount.
func TestRegister(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")

	hub.Register(client)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", hub.ClientCount())
	}

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if !exists {
		t.Error("client not found in hub.clients map")
	}
}

// TestRegisterMultipleClients verifies that multiple clients can be registered.
func TestRegisterMultipleClients(t *testing.T) {
	hub := NewHub(testLogger())

	tests := []struct {
		name   string
		userID string
	}{
		{name: "first client", userID: "user-1"},
		{name: "second client", userID: "user-2"},
		{name: "third client", userID: "user-3"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(tt.userID)
			hub.Register(client)

			wantCount := i + 1
			if hub.ClientCount() != wantCount {
				t.Errorf("ClientCount() = %d, want %d", hub.ClientCount(), wantCount)
			}
		})
	}
}

// TestUnregister verifies that Unregister removes a client and closes its send channel.
func TestUnregister(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")

	hub.Register(client)
	hub.Unregister(client)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", hub.ClientCount())
	}

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if exists {
		t.Error("client still exists in hub.clients map after unregister")
	}

	// Verify channel is closed by attempting to receive.
	_, ok := <-client.send
	if ok {
		t.Error("client.send channel is not closed")
	}
}

// TestUnregisterNotRegistered verifies that Unregister on a client not in the hub does nothing.
func TestUnregisterNotRegistered(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")

	// Unregister without registering first should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unregister() panicked: %v", r)
		}
	}()

	hub.Unregister(client)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", hub.ClientCount())
	}

	// Channel should not be closed if client was never registered.
	select {
	case _, ok := <-client.send:
		if !ok {
			t.Error("channel closed for unregistered client")
		}
	default:
		// Channel is empty and not closed, as expected.
	}
}

// TestBroadcast verifies that Broadcast delivers a message to all registered clients.
func TestBroadcast(t *testing.T) {
	hub := NewHub(testLogger())

	client1 := newTestClient("user-1")
	client2 := newTestClient("user-2")
	client3 := newTestClient("user-3")

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)

	msg := Message{
		Type:      MessageScanStarted,
		ScanID:    "scan-123",
		Timestamp: time.Now(),
		Data:      ScanStartedData{TargetCIDR: "192.168.1.0/24", Status: "running"},
	}

	hub.Broadcast(msg)

	// Verify all clients received the message.
	clients := []*Client{client1, client2, client3}
	for i, client := range clients {
		select {
		case received := <-client.send:
			if received.Type != MessageScanStarted {
				t.Errorf("client %d received Type = %v, want %v", i+1, received.Type, MessageScanStarted)
			}
			if received.ScanID != "scan-123" {
				t.Errorf("client %d received ScanID = %v, want %v", i+1, received.ScanID, "scan-123")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d did not receive message", i+1)
		}
	}
}

// TestBroadcastEmptyHub verifies that Broadcast to empty hub does nothing.
func TestBroadcastEmptyHub(t *testing.T) {
	hub := NewHub(testLogger())

	msg := Message{
		Type:      MessageScanCompleted,
		ScanID:    "scan-123",
		Timestamp: time.Now(),
		Data:      ScanCompletedData{Total: 5, Online: 3, EndedAt: "2026-02-06T12:00:00Z"},
	}

	// Should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Broadcast() to empty hub panicked: %v", r)
		}
	}()

	hub.Broadcast(msg)
}

// TestBroadcastDropsMessagesWhenBufferFull verifies that Broadcast drops messages when client send buffer is full.
func TestBroadcastDropsMessagesWhenBufferFull(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")

	hub.Register(client)

	// Fill the client's send buffer (capacity is 256).
	for i := 0; i < 256; i++ {
		client.send <- Message{
			Type:      MessageScanProgress,
			ScanID:    "scan-fill",
			Timestamp: time.Now(),
		}
	}

	// Verify buffer is full.
	if len(client.send) != 256 {
		t.Fatalf("client.send buffer length = %d, want 256", len(client.send))
	}

	// Broadcast one more message -- should be dropped since buffer is full.
	msg := Message{
		Type:      MessageScanError,
		ScanID:    "scan-dropped",
		Timestamp: time.Now(),
		Data:      ScanErrorData{Error: "test error"},
	}

	hub.Broadcast(msg)

	// The buffer should still be at capacity, and the new message should not be there.
	if len(client.send) != 256 {
		t.Errorf("client.send buffer length = %d, want 256 (message should have been dropped)", len(client.send))
	}

	// Drain one message and verify it's not the dropped message.
	received := <-client.send
	if received.ScanID == "scan-dropped" {
		t.Error("dropped message was unexpectedly received")
	}
}

// TestConcurrentRegisterUnregisterBroadcast verifies that concurrent operations are safe.
func TestConcurrentRegisterUnregisterBroadcast(t *testing.T) {
	hub := NewHub(testLogger())

	var wg sync.WaitGroup
	numClients := 50
	numBroadcasts := 100

	// Concurrently register and unregister clients.
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := newTestClient(string(rune('a' + id)))
			hub.Register(client)

			// Drain messages to prevent buffer from filling.
			go func() {
				for range client.send {
					// Discard messages.
				}
			}()

			time.Sleep(10 * time.Millisecond)
			hub.Unregister(client)
		}(i)
	}

	// Concurrently broadcast messages.
	for i := 0; i < numBroadcasts; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := Message{
				Type:      MessageScanProgress,
				ScanID:    "concurrent-test",
				Timestamp: time.Now(),
				Data:      ScanProgressData{HostsAlive: id, SubnetSize: 256},
			}
			hub.Broadcast(msg)
		}(i)
	}

	wg.Wait()

	// After all goroutines complete, hub should be stable.
	finalCount := hub.ClientCount()
	if finalCount < 0 {
		t.Errorf("ClientCount() = %d, should not be negative", finalCount)
	}
}

// TestConcurrentClientCount verifies that ClientCount is safe to call concurrently.
func TestConcurrentClientCount(t *testing.T) {
	hub := NewHub(testLogger())

	var wg sync.WaitGroup
	var countSum int64

	// Register some clients.
	for i := 0; i < 10; i++ {
		hub.Register(newTestClient(string(rune('a' + i))))
	}

	// Concurrently call ClientCount.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := hub.ClientCount()
			atomic.AddInt64(&countSum, int64(count))
		}()
	}

	wg.Wait()

	// All calls should have returned the same count (10).
	expectedSum := int64(10 * 100)
	if countSum != expectedSum {
		t.Errorf("sum of all ClientCount() calls = %d, want %d", countSum, expectedSum)
	}
}

// TestBroadcastMessageTypes verifies that different message types can be broadcast.
func TestBroadcastMessageTypes(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")
	hub.Register(client)

	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "scan started",
			msg: Message{
				Type:      MessageScanStarted,
				ScanID:    "scan-1",
				Timestamp: time.Now(),
				Data:      ScanStartedData{TargetCIDR: "10.0.0.0/8", Status: "running"},
			},
		},
		{
			name: "scan progress",
			msg: Message{
				Type:      MessageScanProgress,
				ScanID:    "scan-1",
				Timestamp: time.Now(),
				Data:      ScanProgressData{HostsAlive: 42, SubnetSize: 256},
			},
		},
		{
			name: "scan device found",
			msg: Message{
				Type:      MessageScanDeviceFound,
				ScanID:    "scan-1",
				Timestamp: time.Now(),
				Data:      ScanDeviceFoundData{Device: nil}, // Device details not relevant for hub test.
			},
		},
		{
			name: "scan completed",
			msg: Message{
				Type:      MessageScanCompleted,
				ScanID:    "scan-1",
				Timestamp: time.Now(),
				Data:      ScanCompletedData{Total: 10, Online: 8, EndedAt: "2026-02-06T12:00:00Z"},
			},
		},
		{
			name: "scan error",
			msg: Message{
				Type:      MessageScanError,
				ScanID:    "scan-1",
				Timestamp: time.Now(),
				Data:      ScanErrorData{Error: "network unreachable"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub.Broadcast(tt.msg)

			select {
			case received := <-client.send:
				if received.Type != tt.msg.Type {
					t.Errorf("received Type = %v, want %v", received.Type, tt.msg.Type)
				}
				if received.ScanID != tt.msg.ScanID {
					t.Errorf("received ScanID = %v, want %v", received.ScanID, tt.msg.ScanID)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("client did not receive message")
			}
		})
	}
}

// TestClientChannelCapacity verifies that client send channel has correct buffer size.
func TestClientChannelCapacity(t *testing.T) {
	client := newTestClient("user-1")

	if cap(client.send) != 256 {
		t.Errorf("client.send channel capacity = %d, want 256", cap(client.send))
	}
}

// TestUnregisterTwice verifies that unregistering the same client twice is safe.
func TestUnregisterTwice(t *testing.T) {
	hub := NewHub(testLogger())
	client := newTestClient("user-1")

	hub.Register(client)
	hub.Unregister(client)

	// Second unregister should not panic or cause issues.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("second Unregister() panicked: %v", r)
		}
	}()

	hub.Unregister(client)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", hub.ClientCount())
	}
}
