package model

import "time"

// GameID uniquely identifies a game
type GameID string

// GameState represents the current phase of a game
type GameState string

const (
	GameStateAnnouncing GameState = "announcing" // Waiting for announcer to pick letter
	GameStatePlacing    GameState = "placing"    // Players placing the announced letter
	GameStateScoring    GameState = "scoring"    // Game complete, showing scores
	GameStateAbandoned  GameState = "abandoned"  // Game was cancelled
)

// Game represents a single instance of the crossword game
type Game struct {
	ID        GameID
	LobbyCode LobbyCode
	State     GameState
	GridSize  int

	// Players in this game (snapshot at game start)
	Players []PlayerID

	// Turn management
	CurrentTurn   int  // 0-indexed turn number
	AnnouncerIdx  int  // Index into Players for current announcer
	CurrentLetter rune // The letter announced this turn (0 if awaiting)

	// Placement tracking for current turn
	Placements map[PlayerID]bool // Which players have placed this turn

	// Timing
	TurnStartedAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TotalTurns returns the total number of turns in the game (grid cells)
func (g *Game) TotalTurns() int {
	return g.GridSize * g.GridSize
}

// IsComplete returns true if all turns have been played
func (g *Game) IsComplete() bool {
	return g.CurrentTurn >= g.TotalTurns()
}

// CurrentAnnouncer returns the PlayerID of the current announcer
func (g *Game) CurrentAnnouncer() PlayerID {
	if len(g.Players) == 0 {
		return ""
	}
	return g.Players[g.AnnouncerIdx]
}

// AllPlayersPlaced returns true if all players have placed this turn
func (g *Game) AllPlayersPlaced() bool {
	for _, playerID := range g.Players {
		if !g.Placements[playerID] {
			return false
		}
	}
	return true
}

// GameSummary is a lightweight record of a completed game
type GameSummary struct {
	ID          GameID
	FinalScores map[PlayerID]int
	Winner      PlayerID // Empty if tie
	CompletedAt time.Time
}
