package model

import "time"

// EventType identifies the type of event
type EventType string

const (
	// Lobby events
	EventPlayerJoined EventType = "player_joined"
	EventPlayerLeft   EventType = "player_left"
	EventHostChanged  EventType = "host_changed"
	EventRoleChanged  EventType = "role_changed"
	EventGameStarted  EventType = "game_started"
	EventGameEnded    EventType = "game_ended"

	// Game events
	EventLetterAnnounced EventType = "letter_announced"
	EventLetterPlaced    EventType = "letter_placed"
	EventTurnComplete    EventType = "turn_complete"
	EventGameComplete    EventType = "game_complete"
	EventGameAbandoned   EventType = "game_abandoned"
)

// Event is the base structure for all events
type Event struct {
	Type      EventType
	Timestamp time.Time
	LobbyCode LobbyCode
	GameID    GameID   // Empty for lobby-only events
	PlayerID  PlayerID // The player who triggered or is affected
	Payload   any      // Type-specific data
}

// PlayerJoinedPayload contains data for player joined events
type PlayerJoinedPayload struct {
	Player Player
	Role   LobbyMemberRole
}

// PlayerLeftPayload contains data for player left events
type PlayerLeftPayload struct {
	PlayerID    PlayerID
	DisplayName string
}

// HostChangedPayload contains data for host changed events
type HostChangedPayload struct {
	OldHostID PlayerID
	NewHostID PlayerID
}

// RoleChangedPayload contains data for role changed events
type RoleChangedPayload struct {
	PlayerID PlayerID
	OldRole  LobbyMemberRole
	NewRole  LobbyMemberRole
}

// GameStartedPayload contains data for game started events
type GameStartedPayload struct {
	GameID   GameID
	Players  []PlayerID
	GridSize int
}

// LetterAnnouncedPayload contains data for letter announced events
type LetterAnnouncedPayload struct {
	Letter      rune
	AnnouncerID PlayerID
	TurnNumber  int
}

// LetterPlacedPayload contains data for letter placed events
type LetterPlacedPayload struct {
	PlayerID PlayerID
	Position Position
	Letter   rune
}

// TurnCompletePayload contains data for turn complete events
type TurnCompletePayload struct {
	TurnNumber      int
	NextAnnouncerID PlayerID
}

// GameCompletePayload contains data for game complete events
type GameCompletePayload struct {
	Scores []BoardScore
	Winner PlayerID // Empty if tie
}

// GameAbandonedPayload contains data for game abandoned events
type GameAbandonedPayload struct {
	Reason string
}
