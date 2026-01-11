package sse

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

func TestWrapForOOBSwap(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		html     string
		expected string
	}{
		{
			name:     "simple content",
			id:       "member-list",
			html:     "<p>Hello</p>",
			expected: `<div id="member-list" hx-swap-oob="true"><p>Hello</p></div>`,
		},
		{
			name:     "empty content",
			id:       "status",
			html:     "",
			expected: `<div id="status" hx-swap-oob="true"></div>`,
		},
		{
			name:     "complex content",
			id:       "game-board",
			html:     "<div class=\"board\"><button>A</button></div>",
			expected: `<div id="game-board" hx-swap-oob="true"><div class="board"><button>A</button></div></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapForOOBSwap(tt.id, tt.html)
			if result != tt.expected {
				t.Errorf("WrapForOOBSwap(%q, %q)\ngot:  %q\nwant: %q",
					tt.id, tt.html, result, tt.expected)
			}
		})
	}
}

func TestBroadcaster_BroadcastMemberListUpdate(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	// Create a lobby
	lobby := &model.Lobby{
		Code: "ABC123",
		Members: []model.LobbyMember{
			{
				Player: model.Player{ID: "player1", DisplayName: "Alice"},
				Role:   model.RolePlayer,
				IsHost: true,
			},
			{
				Player: model.Player{ID: "player2", DisplayName: "Bob"},
				Role:   model.RolePlayer,
				IsHost: false,
			},
		},
	}

	// Create hub and client
	hub := manager.GetOrCreateHub(lobby.Code)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast member list update
	ctx := context.Background()
	broadcaster.BroadcastMemberListUpdate(ctx, lobby)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		// Should contain the event name
		if !strings.Contains(msgStr, "event: member-update") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain OOB swap wrapper
		if !strings.Contains(msgStr, `hx-swap-oob="true"`) {
			t.Errorf("message does not contain OOB swap: %s", msgStr)
		}
		// Should contain member-list ID
		if !strings.Contains(msgStr, `id="member-list"`) {
			t.Errorf("message does not contain member-list ID: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobby.Code)
}

func TestBroadcaster_BroadcastGameStarted(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("GAME1")

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast game started
	broadcaster.BroadcastGameStarted(lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: game-started") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain a signal (HTMX handles navigation via hx-trigger)
		if !strings.Contains(msgStr, "data: started") {
			t.Errorf("message does not contain started signal: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastLetterAnnounced(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("GAME2")

	// Create a game
	game := &model.Game{
		ID:            "game1",
		CurrentLetter: 'A',
		State:         model.GameStatePlacing,
	}

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast letter announced
	ctx := context.Background()
	broadcaster.BroadcastLetterAnnounced(ctx, game, lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: letter-announced") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain the letter
		if !strings.Contains(msgStr, "A") {
			t.Errorf("message does not contain the letter: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastPlacementUpdate(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("GAME3")

	// Create a game with placements
	game := &model.Game{
		ID:      "game1",
		Players: []model.PlayerID{"player1", "player2", "player3"},
		Placements: map[model.PlayerID]bool{
			"player1": true,
			"player2": true,
			"player3": false,
		},
	}

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast placement update
	ctx := context.Background()
	broadcaster.BroadcastPlacementUpdate(ctx, game, lobbyCode, "player2")

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: placement-update") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain placement count (2/3)
		if !strings.Contains(msgStr, "2/3") {
			t.Errorf("message does not contain placement count: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastTurnComplete(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("GAME4")
	game := &model.Game{ID: "game1", CurrentTurn: 3}

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast turn complete
	ctx := context.Background()
	broadcaster.BroadcastTurnComplete(ctx, game, lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: turn-complete") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain turn number (client JS handles reload)
		if !strings.Contains(msgStr, "data: 3") {
			t.Errorf("message does not contain turn number: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastGameComplete(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("GAME5")

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast game complete
	broadcaster.BroadcastGameComplete(lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: game-complete") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain simple signal (client JS handles reload)
		if !strings.Contains(msgStr, "data: complete") {
			t.Errorf("message does not contain complete signal: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastRefresh(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("REFRESH")

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast refresh
	broadcaster.BroadcastRefresh(lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: refresh") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain simple signal (client JS handles reload)
		if !strings.Contains(msgStr, "data: refresh") {
			t.Errorf("message does not contain refresh signal: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_BroadcastGameAbandoned(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	lobbyCode := model.LobbyCode("ABANDON")

	// Create hub and client
	hub := manager.GetOrCreateHub(lobbyCode)
	client := NewClient(hub, "player1")
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast game abandoned
	broadcaster.BroadcastGameAbandoned(lobbyCode)

	// Verify client received the message
	select {
	case msg := <-client.send:
		msgStr := string(msg)
		if !strings.Contains(msgStr, "event: game-abandoned") {
			t.Errorf("message does not contain event name: %s", msgStr)
		}
		// Should contain simple signal (HTMX handles navigation)
		if !strings.Contains(msgStr, "data: abandoned") {
			t.Errorf("message does not contain abandoned signal: %s", msgStr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}

	manager.RemoveHub(lobbyCode)
}

func TestBroadcaster_NoHubDoesNotPanic(t *testing.T) {
	manager := NewHubManager()
	broadcaster := NewBroadcaster(manager)

	// These should not panic when hub doesn't exist
	ctx := context.Background()
	lobby := &model.Lobby{Code: "NOEXIST", Members: []model.LobbyMember{}}
	game := &model.Game{ID: "game1"}

	broadcaster.BroadcastMemberListUpdate(ctx, lobby)
	broadcaster.BroadcastGameStarted("NOEXIST")
	broadcaster.BroadcastLetterAnnounced(ctx, game, "NOEXIST")
	broadcaster.BroadcastPlacementUpdate(ctx, game, "NOEXIST", "player1")
	broadcaster.BroadcastTurnComplete(ctx, game, "NOEXIST")
	broadcaster.BroadcastGameComplete("NOEXIST")
	broadcaster.BroadcastGameAbandoned("NOEXIST")
	broadcaster.BroadcastRefresh("NOEXIST")

	// If we get here without panic, test passed
}
