package sse

import (
	"log/slog"
	"sync"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// Hub manages SSE clients for a single lobby
type Hub struct {
	lobbyCode model.LobbyCode
	clients   map[*Client]bool
	mu        sync.RWMutex
	logger    *slog.Logger

	// Channels for managing clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	done       chan struct{}
}

// NewHub creates a new Hub for a lobby
func NewHub(lobbyCode model.LobbyCode, logger *slog.Logger) *Hub {
	return &Hub{
		lobbyCode:  lobbyCode,
		clients:    make(map[*Client]bool),
		logger:     logger.With(slog.String("lobby", string(lobbyCode))),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	h.logger.Info("sse hub started")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mu.Unlock()
			h.logger.Info("sse client registered",
				slog.String("player_id", string(client.playerID)),
				slog.Int("total_clients", clientCount))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				clientCount := len(h.clients)
				h.mu.Unlock()
				duration := time.Since(client.connectedAt)
				h.logger.Info("sse client unregistered",
					slog.String("player_id", string(client.playerID)),
					slog.Duration("connection_duration", duration),
					slog.Int("total_clients", clientCount))
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			sentCount := 0
			droppedCount := 0
			for client := range h.clients {
				select {
				case client.send <- message:
					sentCount++
				default:
					droppedCount++
					h.logger.Warn("sse message dropped - client buffer full",
						slog.String("player_id", string(client.playerID)))
				}
			}
			h.mu.RUnlock()
			if droppedCount > 0 {
				h.logger.Warn("sse broadcast partial failure",
					slog.Int("sent", sentCount),
					slog.Int("dropped", droppedCount))
			}

		case <-h.done:
			h.mu.Lock()
			clientCount := len(h.clients)
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			h.logger.Info("sse hub stopped", slog.Int("disconnected_clients", clientCount))
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
		h.logger.Warn("sse broadcast dropped - hub buffer full")
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
// Multi-line data is properly formatted with "data: " prefix on each line
func formatSSEMessage(eventName, data string) []byte {
	msg := "event: " + eventName + "\n"
	// SSE requires each line of data to be prefixed with "data: "
	lines := splitLines(data)
	for _, line := range lines {
		msg += "data: " + line + "\n"
	}
	msg += "\n"
	return []byte(msg)
}

// splitLines splits a string into lines, handling various line endings
func splitLines(s string) []string {
	var lines []string
	var current string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else if r != '\r' {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	return lines
}

// HubManager manages hubs for all lobbies
type HubManager struct {
	hubs   map[model.LobbyCode]*Hub
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewHubManager creates a new HubManager
func NewHubManager(logger *slog.Logger) *HubManager {
	return &HubManager{
		hubs:   make(map[model.LobbyCode]*Hub),
		logger: logger.With(slog.String("component", "sse")),
	}
}

// GetOrCreateHub returns the hub for a lobby, creating one if it doesn't exist
func (m *HubManager) GetOrCreateHub(lobbyCode model.LobbyCode) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[lobbyCode]; ok {
		return hub
	}

	hub := NewHub(lobbyCode, m.logger)
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
		m.logger.Info("sse hub removed", slog.String("lobby", string(lobbyCode)))
	}
}

// CleanupEmptyHubs removes hubs with no clients
func (m *HubManager) CleanupEmptyHubs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	removedCount := 0
	for code, hub := range m.hubs {
		if hub.ClientCount() == 0 {
			hub.Close()
			delete(m.hubs, code)
			removedCount++
		}
	}
	if removedCount > 0 {
		m.logger.Info("sse empty hubs cleaned up", slog.Int("removed", removedCount))
	}
}
