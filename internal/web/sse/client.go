package sse

import (
	"net/http"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time between keepalive pings
	pingPeriod = 30 * time.Second

	// Buffer size for outgoing messages
	sendBufferSize = 256
)

// Client represents a connected SSE client
type Client struct {
	hub      *Hub
	playerID model.PlayerID
	send     chan []byte
}

// NewClient creates a new SSE client
func NewClient(hub *Hub, playerID model.PlayerID) *Client {
	return &Client{
		hub:      hub,
		playerID: playerID,
		send:     make(chan []byte, sendBufferSize),
	}
}

// ServeSSE handles the SSE connection for a client
func ServeSSE(w http.ResponseWriter, r *http.Request, hub *Hub, playerID model.PlayerID) {
	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create and register client
	client := NewClient(hub, playerID)
	hub.Register(client)

	// Ensure cleanup on disconnect
	defer func() {
		hub.Unregister(client)
	}()

	// Send initial connection event
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
	flusher.Flush()

	// Create ticker for keepalive
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Handle client connection
	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// Hub closed the channel
				return
			}
			_, err := w.Write(message)
			if err != nil {
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// Send keepalive comment
			_, err := w.Write([]byte(": keepalive\n\n"))
			if err != nil {
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}
