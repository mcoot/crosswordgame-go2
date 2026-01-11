package sse

import (
	"testing"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

func TestFormatSSEMessage(t *testing.T) {
	tests := []struct {
		name      string
		eventName string
		data      string
		expected  string
	}{
		{
			name:      "single line data",
			eventName: "test-event",
			data:      "hello world",
			expected:  "event: test-event\ndata: hello world\n\n",
		},
		{
			name:      "multi-line data",
			eventName: "member-update",
			data:      "<div>\n  <p>line1</p>\n  <p>line2</p>\n</div>",
			expected:  "event: member-update\ndata: <div>\ndata:   <p>line1</p>\ndata:   <p>line2</p>\ndata: </div>\n\n",
		},
		{
			name:      "empty data",
			eventName: "ping",
			data:      "",
			expected:  "event: ping\ndata: \n\n",
		},
		{
			name:      "data with carriage returns",
			eventName: "test",
			data:      "line1\r\nline2",
			expected:  "event: test\ndata: line1\ndata: line2\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSSEMessage(tt.eventName, tt.data)
			if string(result) != tt.expected {
				t.Errorf("formatSSEMessage(%q, %q)\ngot:  %q\nwant: %q",
					tt.eventName, tt.data, string(result), tt.expected)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single line",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "two lines",
			input:    "line1\nline2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "trailing newline",
			input:    "line1\n",
			expected: []string{"line1"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "crlf line endings",
			input:    "line1\r\nline2\r\n",
			expected: []string{"line1", "line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitLines(%q) returned %d lines, want %d",
					tt.input, len(result), len(tt.expected))
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q",
						tt.input, i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub("TESTCODE")
	go hub.Run()
	defer hub.Close()

	// Create a client
	client := NewClient(hub, "player1")
	hub.Register(client)

	// Give the hub time to process registration
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", hub.ClientCount())
	}

	// Broadcast a message
	hub.BroadcastEvent("test-event", "test data")

	// Client should receive the message
	select {
	case msg := <-client.send:
		expected := "event: test-event\ndata: test data\n\n"
		if string(msg) != expected {
			t.Errorf("client received %q, want %q", string(msg), expected)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub("TESTCODE")
	go hub.Run()
	defer hub.Close()

	client := NewClient(hub, "player1")
	hub.Register(client)

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", hub.ClientCount())
	}

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d after unregister, want 0", hub.ClientCount())
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub("TESTCODE")
	go hub.Run()
	defer hub.Close()

	// Create multiple clients
	client1 := NewClient(hub, "player1")
	client2 := NewClient(hub, "player2")
	client3 := NewClient(hub, "player3")

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("ClientCount() = %d, want 3", hub.ClientCount())
	}

	// Broadcast a message
	hub.BroadcastEvent("update", "data")

	// All clients should receive the message
	for i, client := range []*Client{client1, client2, client3} {
		select {
		case msg := <-client.send:
			expected := "event: update\ndata: data\n\n"
			if string(msg) != expected {
				t.Errorf("client %d received %q, want %q", i+1, string(msg), expected)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d did not receive message", i+1)
		}
	}
}

func TestHubManager_GetOrCreateHub(t *testing.T) {
	manager := NewHubManager()

	// Get or create a hub
	hub1 := manager.GetOrCreateHub("ABC123")
	if hub1 == nil {
		t.Fatal("GetOrCreateHub returned nil")
	}

	// Getting again should return the same hub
	hub2 := manager.GetOrCreateHub("ABC123")
	if hub1 != hub2 {
		t.Error("GetOrCreateHub returned different hub for same code")
	}

	// Different code should return different hub
	hub3 := manager.GetOrCreateHub("XYZ789")
	if hub3 == hub1 {
		t.Error("GetOrCreateHub returned same hub for different code")
	}

	// Clean up
	manager.RemoveHub("ABC123")
	manager.RemoveHub("XYZ789")
}

func TestHubManager_GetHub(t *testing.T) {
	manager := NewHubManager()

	// GetHub on non-existent hub should return nil
	hub := manager.GetHub("NOTEXIST")
	if hub != nil {
		t.Error("GetHub returned non-nil for non-existent hub")
	}

	// Create a hub then get it
	created := manager.GetOrCreateHub("ABC123")
	got := manager.GetHub("ABC123")
	if got != created {
		t.Error("GetHub returned different hub than GetOrCreateHub")
	}

	manager.RemoveHub("ABC123")
}

func TestHubManager_RemoveHub(t *testing.T) {
	manager := NewHubManager()

	hub := manager.GetOrCreateHub("ABC123")
	_ = hub // Just to ensure it's created

	manager.RemoveHub("ABC123")

	// Hub should be gone
	got := manager.GetHub("ABC123")
	if got != nil {
		t.Error("Hub still exists after RemoveHub")
	}

	// Removing non-existent hub should not panic
	manager.RemoveHub("NOTEXIST")
}

func TestHubManager_CleanupEmptyHubs(t *testing.T) {
	manager := NewHubManager()

	// Create a hub with no clients
	hub1 := manager.GetOrCreateHub(model.LobbyCode("EMPTY"))
	_ = hub1

	// Create a hub with a client
	hub2 := manager.GetOrCreateHub(model.LobbyCode("ACTIVE"))
	client := NewClient(hub2, "player1")
	hub2.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Cleanup empty hubs
	manager.CleanupEmptyHubs()

	// Empty hub should be gone
	if manager.GetHub("EMPTY") != nil {
		t.Error("Empty hub still exists after cleanup")
	}

	// Active hub should still exist
	if manager.GetHub("ACTIVE") == nil {
		t.Error("Active hub was removed during cleanup")
	}

	manager.RemoveHub("ACTIVE")
}
