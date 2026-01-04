package model

import "time"

// LobbyCode is a human-readable identifier for joining lobbies
type LobbyCode string

// LobbyState represents the current state of a lobby
type LobbyState string

const (
	LobbyStateWaiting LobbyState = "waiting" // No game in progress
	LobbyStateInGame  LobbyState = "in_game" // Game currently active
)

// LobbyMemberRole distinguishes players from spectators
type LobbyMemberRole string

const (
	RolePlayer    LobbyMemberRole = "player"
	RoleSpectator LobbyMemberRole = "spectator"
)

// LobbyMember represents a player's membership in a lobby
type LobbyMember struct {
	Player   Player
	Role     LobbyMemberRole
	IsHost   bool
	JoinedAt time.Time
}

// LobbyConfig holds configurable settings for games in this lobby
type LobbyConfig struct {
	GridSize int // Default 5, configurable
}

// DefaultLobbyConfig returns the default lobby configuration
func DefaultLobbyConfig() LobbyConfig {
	return LobbyConfig{
		GridSize: 5,
	}
}

// Lobby represents a group of players who can play games together
type Lobby struct {
	Code        LobbyCode
	State       LobbyState
	Members     []LobbyMember // All members (players + spectators)
	Config      LobbyConfig
	GameHistory []GameSummary // Completed games
	CurrentGame *GameID       // nil when State is waiting
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GetHost returns the current host member, or nil if none
func (l *Lobby) GetHost() *LobbyMember {
	for i := range l.Members {
		if l.Members[i].IsHost {
			return &l.Members[i]
		}
	}
	return nil
}

// GetMember returns the member with the given player ID, or nil if not found
func (l *Lobby) GetMember(playerID PlayerID) *LobbyMember {
	for i := range l.Members {
		if l.Members[i].Player.ID == playerID {
			return &l.Members[i]
		}
	}
	return nil
}

// GetPlayers returns all members with the player role
func (l *Lobby) GetPlayers() []LobbyMember {
	var players []LobbyMember
	for _, m := range l.Members {
		if m.Role == RolePlayer {
			players = append(players, m)
		}
	}
	return players
}

// GetSpectators returns all members with the spectator role
func (l *Lobby) GetSpectators() []LobbyMember {
	var spectators []LobbyMember
	for _, m := range l.Members {
		if m.Role == RoleSpectator {
			spectators = append(spectators, m)
		}
	}
	return spectators
}
