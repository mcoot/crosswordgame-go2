package redis

import "time"

// Config holds Redis connection and behavior settings
type Config struct {
	// URL is the Redis connection URL (e.g., redis://localhost:6379)
	URL string

	// Pool settings
	PoolSize     int
	MinIdleConns int

	// TTL settings for different entity types
	GuestPlayerTTL time.Duration
	LobbyTTL       time.Duration
	GameTTL        time.Duration
	BoardTTL       time.Duration
}

// DefaultConfig returns sensible defaults for Redis configuration
func DefaultConfig() Config {
	return Config{
		URL:            "redis://localhost:6379",
		PoolSize:       10,
		MinIdleConns:   2,
		GuestPlayerTTL: 24 * time.Hour,
		LobbyTTL:       24 * time.Hour,
		GameTTL:        24 * time.Hour,
		BoardTTL:       24 * time.Hour,
	}
}
