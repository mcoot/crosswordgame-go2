package model

import "time"

// PlayerID uniquely identifies a player across the system
type PlayerID string

// Player represents a game participant
type Player struct {
	ID          PlayerID
	DisplayName string
	IsGuest     bool // true for unregistered players
	CreatedAt   time.Time
}

// RegisteredPlayer extends Player with authentication data
// Stored separately for security (password never in memory with session)
type RegisteredPlayer struct {
	PlayerID     PlayerID
	Username     string // login username (immutable)
	PasswordHash string // bcrypt hash
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
