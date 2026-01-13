package game

import (
	"context"
	"log/slog"
	"unicode"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Controller manages game state machine and turn flow
type Controller struct {
	storage        storage.Storage
	boardService   *board.Service
	scoringService *scoring.Service
	clock          clock.Clock
	random         random.Random
	logger         *slog.Logger
}

// NewController creates a new GameController
func NewController(
	storage storage.Storage,
	boardService *board.Service,
	scoringService *scoring.Service,
	clock clock.Clock,
	random random.Random,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		storage:        storage,
		boardService:   boardService,
		scoringService: scoringService,
		clock:          clock,
		random:         random,
		logger:         logger,
	}
}

// CreateGame initializes a new game with the given players
func (c *Controller) CreateGame(ctx context.Context, lobbyCode model.LobbyCode, players []model.PlayerID, gridSize int) (*model.Game, error) {
	if len(players) == 0 {
		return nil, model.ErrInsufficientPlayers
	}

	now := c.clock.Now()
	gameID := model.GameID(c.random.String(12, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"))

	game := &model.Game{
		ID:            gameID,
		LobbyCode:     lobbyCode,
		State:         model.GameStateAnnouncing,
		GridSize:      gridSize,
		Players:       players,
		CurrentTurn:   0,
		AnnouncerIdx:  0,
		CurrentLetter: 0,
		Placements:    make(map[model.PlayerID]bool),
		TurnStartedAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Create boards for all players
	for _, playerID := range players {
		if _, err := c.boardService.CreateBoard(ctx, gameID, playerID, gridSize); err != nil {
			return nil, err
		}
	}

	if err := c.storage.SaveGame(ctx, game); err != nil {
		c.logger.Error("failed to save game",
			slog.String("game_id", string(game.ID)),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	c.logger.Info("game created",
		slog.String("game_id", string(gameID)),
		slog.String("lobby_code", string(lobbyCode)),
		slog.Int("player_count", len(players)),
		slog.Int("grid_size", gridSize),
	)

	return game, nil
}

// GetGame retrieves a game by ID
func (c *Controller) GetGame(ctx context.Context, gameID model.GameID) (*model.Game, error) {
	return c.storage.GetGame(ctx, gameID)
}

// AnnounceLetter handles the announcer selecting a letter for the turn
func (c *Controller) AnnounceLetter(ctx context.Context, gameID model.GameID, playerID model.PlayerID, letter rune) error {
	game, err := c.storage.GetGame(ctx, gameID)
	if err != nil {
		return err
	}

	// Validate game state
	if game.State == model.GameStateScoring {
		return model.ErrGameComplete
	}
	if game.State == model.GameStateAbandoned {
		return model.ErrGameAbandoned
	}
	if game.State != model.GameStateAnnouncing {
		return model.ErrNotPlayerTurn
	}

	// Validate it's this player's turn to announce
	if game.CurrentAnnouncer() != playerID {
		return model.ErrNotPlayerTurn
	}

	// Validate letter
	if err := board.ValidateLetter(letter); err != nil {
		return err
	}

	// Update game state
	game.CurrentLetter = unicode.ToUpper(letter)
	game.State = model.GameStatePlacing
	game.Placements = make(map[model.PlayerID]bool)
	game.UpdatedAt = c.clock.Now()

	return c.storage.SaveGame(ctx, game)
}

// PlaceLetter handles a player placing the announced letter on their board
func (c *Controller) PlaceLetter(ctx context.Context, gameID model.GameID, playerID model.PlayerID, pos model.Position) error {
	game, err := c.storage.GetGame(ctx, gameID)
	if err != nil {
		return err
	}

	// Validate game state
	if game.State == model.GameStateScoring {
		return model.ErrGameComplete
	}
	if game.State == model.GameStateAbandoned {
		return model.ErrGameAbandoned
	}
	if game.State != model.GameStatePlacing {
		return model.ErrLetterNotAnnounced
	}

	// Validate player is in game
	isPlayer := false
	for _, p := range game.Players {
		if p == playerID {
			isPlayer = true
			break
		}
	}
	if !isPlayer {
		return model.ErrPlayerNotFound
	}

	// Check if already placed
	if game.Placements[playerID] {
		return model.ErrAlreadyPlaced
	}

	// Get and update board
	boardObj, err := c.boardService.GetBoard(ctx, gameID, playerID)
	if err != nil {
		return err
	}

	if err := c.boardService.PlaceLetter(ctx, boardObj, game.CurrentLetter, pos); err != nil {
		return err
	}

	// Mark as placed
	game.Placements[playerID] = true
	game.UpdatedAt = c.clock.Now()

	// Check if all players have placed
	if game.AllPlayersPlaced() {
		return c.advanceTurn(ctx, game)
	}

	return c.storage.SaveGame(ctx, game)
}

// advanceTurn moves to the next turn or completes the game
func (c *Controller) advanceTurn(ctx context.Context, game *model.Game) error {
	game.CurrentTurn++

	if game.CurrentTurn >= game.TotalTurns() {
		// Game complete - move to scoring
		game.State = model.GameStateScoring
		c.logger.Info("game completed",
			slog.String("game_id", string(game.ID)),
			slog.String("lobby_code", string(game.LobbyCode)),
			slog.Int("total_turns", game.CurrentTurn),
		)
	} else {
		// Next turn - rotate announcer
		game.AnnouncerIdx = (game.AnnouncerIdx + 1) % len(game.Players)
		game.State = model.GameStateAnnouncing
		game.CurrentLetter = 0
		game.Placements = make(map[model.PlayerID]bool)
		game.TurnStartedAt = c.clock.Now()
	}

	game.UpdatedAt = c.clock.Now()
	return c.storage.SaveGame(ctx, game)
}

// AbandonGame ends a game prematurely
func (c *Controller) AbandonGame(ctx context.Context, gameID model.GameID) error {
	game, err := c.storage.GetGame(ctx, gameID)
	if err != nil {
		return err
	}

	if game.State == model.GameStateScoring || game.State == model.GameStateAbandoned {
		return nil // Already finished
	}

	game.State = model.GameStateAbandoned
	game.UpdatedAt = c.clock.Now()

	c.logger.Info("game abandoned",
		slog.String("game_id", string(gameID)),
		slog.String("lobby_code", string(game.LobbyCode)),
	)

	return c.storage.SaveGame(ctx, game)
}

// RemovePlayer handles a player leaving mid-game
func (c *Controller) RemovePlayer(ctx context.Context, gameID model.GameID, playerID model.PlayerID) error {
	game, err := c.storage.GetGame(ctx, gameID)
	if err != nil {
		return err
	}

	if game.State == model.GameStateScoring || game.State == model.GameStateAbandoned {
		return nil // Game already finished
	}

	// Find and remove player
	playerIdx := -1
	for i, p := range game.Players {
		if p == playerID {
			playerIdx = i
			break
		}
	}

	if playerIdx == -1 {
		return nil // Player not in game
	}

	// Remove player from list
	game.Players = append(game.Players[:playerIdx], game.Players[playerIdx+1:]...)

	// Check if game should be abandoned (not enough players)
	if len(game.Players) == 0 {
		game.State = model.GameStateAbandoned
		game.UpdatedAt = c.clock.Now()
		return c.storage.SaveGame(ctx, game)
	}

	// Adjust announcer index if needed
	if game.AnnouncerIdx >= len(game.Players) {
		game.AnnouncerIdx = 0
	}

	// If removed player was supposed to announce, skip to placing or next turn
	// (In placing state, mark them as having placed)
	if game.State == model.GameStatePlacing {
		delete(game.Placements, playerID)
		// Check if now all remaining players have placed
		if game.AllPlayersPlaced() {
			return c.advanceTurn(ctx, game)
		}
	}

	game.UpdatedAt = c.clock.Now()
	return c.storage.SaveGame(ctx, game)
}

// GetFinalScores calculates and returns the final scores for a completed game
func (c *Controller) GetFinalScores(ctx context.Context, gameID model.GameID) ([]model.BoardScore, error) {
	game, err := c.storage.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if game.State != model.GameStateScoring {
		return nil, model.ErrNoGameInProgress
	}

	boards, err := c.boardService.GetBoardsForGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	return c.scoringService.ScoreMultipleBoards(boards), nil
}

// CreateGameSummary creates a summary record for a completed game
func (c *Controller) CreateGameSummary(ctx context.Context, gameID model.GameID) (*model.GameSummary, error) {
	scores, err := c.GetFinalScores(ctx, gameID)
	if err != nil {
		return nil, err
	}

	finalScores := make(map[model.PlayerID]int)
	for _, s := range scores {
		finalScores[s.PlayerID] = s.TotalScore
	}

	return &model.GameSummary{
		ID:          gameID,
		FinalScores: finalScores,
		Winner:      c.scoringService.DetermineWinner(scores),
		CompletedAt: c.clock.Now(),
	}, nil
}

// Interface for dependency injection
type ControllerInterface interface {
	CreateGame(ctx context.Context, lobbyCode model.LobbyCode, players []model.PlayerID, gridSize int) (*model.Game, error)
	GetGame(ctx context.Context, gameID model.GameID) (*model.Game, error)
	AnnounceLetter(ctx context.Context, gameID model.GameID, playerID model.PlayerID, letter rune) error
	PlaceLetter(ctx context.Context, gameID model.GameID, playerID model.PlayerID, pos model.Position) error
	AbandonGame(ctx context.Context, gameID model.GameID) error
	RemovePlayer(ctx context.Context, gameID model.GameID, playerID model.PlayerID) error
	GetFinalScores(ctx context.Context, gameID model.GameID) ([]model.BoardScore, error)
	CreateGameSummary(ctx context.Context, gameID model.GameID) (*model.GameSummary, error)
}

var _ ControllerInterface = (*Controller)(nil)
