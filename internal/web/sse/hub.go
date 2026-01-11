package sse

import (
	"sync"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// Hub manages SSE clients for a single lobby
type Hub struct {
	lobbyCode model.LobbyCode
	clients   map[*Client]bool
	mu        sync.RWMutex

	// Channels for managing clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	done       chan struct{}
}

// NewHub creates a new Hub for a lobby
func NewHub(lobbyCode model.LobbyCode) *Hub {
	return &Hub{
		lobbyCode:  lobbyCode,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, will be cleaned up
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		// Drop message if buffer full
	}
}

// BroadcastEvent sends an SSE event with a name and data
func (h *Hub) BroadcastEvent(eventName, data string) {
	msg := formatSSEMessage(eventName, data)
	h.Broadcast(msg)
}

// Close shuts down the hub
func (h *Hub) Close() {
	close(h.done)
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// formatSSEMessage formats an SSE message with event name and data
func formatSSEMessage(eventName, data string) []byte {
	msg := "event: " + eventName + "\n"
	msg += "data: " + data + "\n\n"
	return []byte(msg)
}

// HubManager manages hubs for all lobbies
type HubManager struct {
	hubs map[model.LobbyCode]*Hub
	mu   sync.RWMutex
}

// NewHubManager creates a new HubManager
func NewHubManager() *HubManager {
	return &HubManager{
		hubs: make(map[model.LobbyCode]*Hub),
	}
}

// GetOrCreateHub returns the hub for a lobby, creating one if it doesn't exist
func (m *HubManager) GetOrCreateHub(lobbyCode model.LobbyCode) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[lobbyCode]; ok {
		return hub
	}

	hub := NewHub(lobbyCode)
	m.hubs[lobbyCode] = hub
	go hub.Run()
	return hub
}

// GetHub returns the hub for a lobby, or nil if it doesn't exist
func (m *HubManager) GetHub(lobbyCode model.LobbyCode) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[lobbyCode]
}

// RemoveHub removes and closes a hub
func (m *HubManager) RemoveHub(lobbyCode model.LobbyCode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[lobbyCode]; ok {
		hub.Close()
		delete(m.hubs, lobbyCode)
	}
}

// CleanupEmptyHubs removes hubs with no clients
func (m *HubManager) CleanupEmptyHubs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for code, hub := range m.hubs {
		if hub.ClientCount() == 0 {
			hub.Close()
			delete(m.hubs, code)
		}
	}
}
