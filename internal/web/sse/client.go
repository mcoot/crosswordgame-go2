package sse

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

const (
	// Time between keepalive pings - reduced from 30s to 15s for better connection detection
	pingPeriod = 15 * time.Second

	// Buffer size for outgoing messages
	sendBufferSize = 256
)

// Client represents a connected SSE client
type Client struct {
	hub         *Hub
	playerID    model.PlayerID
	send        chan []byte
	connectedAt time.Time
}

// NewClient creates a new SSE client
func NewClient(hub *Hub, playerID model.PlayerID) *Client {
	return &Client{
		hub:         hub,
		playerID:    playerID,
		send:        make(chan []byte, sendBufferSize),
		connectedAt: time.Now(),
	}
}

// writeDeadlineExtension is the duration to extend the write deadline before each write.
// This must be longer than pingPeriod to ensure the deadline doesn't expire between keepalives.
const writeDeadlineExtension = 30 * time.Second

// ServeSSE handles the SSE connection for a client
func ServeSSE(w http.ResponseWriter, r *http.Request, hub *Hub, playerID model.PlayerID) {
	logger := hub.logger.With(slog.String("player_id", string(playerID)))

	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Error("sse streaming not supported by response writer")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create response controller for deadline management (Go 1.20+)
	// This allows us to extend the write deadline before each write,
	// working around Go's WriteTimeout being an absolute deadline rather than per-write.
	rc := http.NewResponseController(w)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create and register client
	client := NewClient(hub, playerID)
	hub.Register(client)
	defer hub.Unregister(client)

	// Helper to write with deadline extension
	writeWithDeadline := func(data []byte) error {
		// Extend write deadline before each write to prevent timeout
		if err := rc.SetWriteDeadline(time.Now().Add(writeDeadlineExtension)); err != nil {
			logger.Warn("sse failed to set write deadline", slog.Any("error", err))
		}
		_, err := w.Write(data)
		return err
	}

	// Send retry interval to tell client how quickly to reconnect on disconnect
	if err := writeWithDeadline([]byte("retry: 3000\n\n")); err != nil {
		logger.Error("sse failed to write retry header", slog.Any("error", err))
		return
	}
	flusher.Flush()

	// Send initial connection event
	if err := writeWithDeadline([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n")); err != nil {
		logger.Error("sse failed to write connected event", slog.Any("error", err))
		return
	}
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
				logger.Debug("sse client channel closed by hub")
				return
			}
			if err := writeWithDeadline(message); err != nil {
				logger.Warn("sse write error", slog.Any("error", err))
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// Send keepalive comment
			if err := writeWithDeadline([]byte(": keepalive\n\n")); err != nil {
				logger.Warn("sse keepalive write error", slog.Any("error", err))
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			logger.Debug("sse client context done", slog.Any("reason", r.Context().Err()))
			return
		}
	}
}
