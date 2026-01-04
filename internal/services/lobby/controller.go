package lobby

import (
	"context"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

const (
	// LobbyCodeLength is the length of generated lobby codes
	LobbyCodeLength = 6
	// LobbyCodeAlphabet is the characters used in lobby codes (avoid confusing chars)
	LobbyCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// Controller manages lobby state machine and member operations
type Controller struct {
	storage        storage.Storage
	gameController *game.Controller
	clock          clock.Clock
	random         random.Random
}

// NewController creates a new LobbyController
func NewController(
	storage storage.Storage,
	gameController *game.Controller,
	clock clock.Clock,
	random random.Random,
) *Controller {
	return &Controller{
		storage:        storage,
		gameController: gameController,
		clock:          clock,
		random:         random,
	}
}

// CreateLobby creates a new lobby with the given player as host
func (c *Controller) CreateLobby(ctx context.Context, host model.Player) (*model.Lobby, error) {
	now := c.clock.Now()

	// Generate unique lobby code
	var code model.LobbyCode
	for {
		code = model.LobbyCode(c.random.String(LobbyCodeLength, LobbyCodeAlphabet))
		exists, err := c.storage.LobbyExists(ctx, code)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
	}

	lobby := &model.Lobby{
		Code:   code,
		State:  model.LobbyStateWaiting,
		Config: model.DefaultLobbyConfig(),
		Members: []model.LobbyMember{
			{
				Player:   host,
				Role:     model.RolePlayer,
				IsHost:   true,
				JoinedAt: now,
			},
		},
		GameHistory: []model.GameSummary{},
		CurrentGame: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := c.storage.SaveLobby(ctx, lobby); err != nil {
		return nil, err
	}

	return lobby, nil
}

// GetLobby retrieves a lobby by code
func (c *Controller) GetLobby(ctx context.Context, code model.LobbyCode) (*model.Lobby, error) {
	return c.storage.GetLobby(ctx, code)
}

// JoinLobby adds a player to a lobby
func (c *Controller) JoinLobby(ctx context.Context, code model.LobbyCode, player model.Player) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Check if already in lobby
	if lobby.GetMember(player.ID) != nil {
		return model.ErrAlreadyInLobby
	}

	// Determine role - spectator if game in progress, player otherwise
	role := model.RolePlayer
	if lobby.State == model.LobbyStateInGame {
		role = model.RoleSpectator
	}

	lobby.Members = append(lobby.Members, model.LobbyMember{
		Player:   player,
		Role:     role,
		IsHost:   false,
		JoinedAt: c.clock.Now(),
	})
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// LeaveLobby removes a player from a lobby
func (c *Controller) LeaveLobby(ctx context.Context, code model.LobbyCode, playerID model.PlayerID) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	member := lobby.GetMember(playerID)
	if member == nil {
		return model.ErrNotInLobby
	}

	wasHost := member.IsHost
	wasPlayer := member.Role == model.RolePlayer

	// Remove member
	for i, m := range lobby.Members {
		if m.Player.ID == playerID {
			lobby.Members = append(lobby.Members[:i], lobby.Members[i+1:]...)
			break
		}
	}

	// If lobby is now empty, delete it
	if len(lobby.Members) == 0 {
		// Abandon any current game first
		if lobby.CurrentGame != nil {
			_ = c.gameController.AbandonGame(ctx, *lobby.CurrentGame)
		}
		return c.storage.DeleteLobby(ctx, code)
	}

	// If host left, assign new host
	if wasHost {
		lobby.Members[0].IsHost = true
	}

	// If player left during game, remove from game
	if wasPlayer && lobby.CurrentGame != nil {
		if err := c.gameController.RemovePlayer(ctx, *lobby.CurrentGame, playerID); err != nil {
			// Check if game was abandoned due to no players
			g, _ := c.gameController.GetGame(ctx, *lobby.CurrentGame)
			if g != nil && g.State == model.GameStateAbandoned {
				lobby.State = model.LobbyStateWaiting
				lobby.CurrentGame = nil
			}
		}
	}

	lobby.UpdatedAt = c.clock.Now()
	return c.storage.SaveLobby(ctx, lobby)
}

// SetRole changes a member's role (player/spectator)
func (c *Controller) SetRole(ctx context.Context, code model.LobbyCode, playerID model.PlayerID, role model.LobbyMemberRole) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Cannot change roles during a game
	if lobby.State == model.LobbyStateInGame {
		return model.ErrGameInProgress
	}

	member := lobby.GetMember(playerID)
	if member == nil {
		return model.ErrNotInLobby
	}

	member.Role = role
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// TransferHost makes another member the host
func (c *Controller) TransferHost(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID, newHostID model.PlayerID) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Verify requester is current host
	currentHost := lobby.GetHost()
	if currentHost == nil || currentHost.Player.ID != requestingPlayer {
		return model.ErrNotHost
	}

	// Verify new host is in lobby
	newHost := lobby.GetMember(newHostID)
	if newHost == nil {
		return model.ErrNotInLobby
	}

	// Transfer host
	currentHost.IsHost = false
	newHost.IsHost = true
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// StartGame begins a new game with current players
func (c *Controller) StartGame(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID) (*model.Game, error) {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return nil, err
	}

	// Verify requester is host
	host := lobby.GetHost()
	if host == nil || host.Player.ID != requestingPlayer {
		return nil, model.ErrNotHost
	}

	// Cannot start if game in progress
	if lobby.State == model.LobbyStateInGame {
		return nil, model.ErrGameInProgress
	}

	// Get players (not spectators)
	players := lobby.GetPlayers()
	if len(players) == 0 {
		return nil, model.ErrInsufficientPlayers
	}

	// Extract player IDs
	playerIDs := make([]model.PlayerID, len(players))
	for i, p := range players {
		playerIDs[i] = p.Player.ID
	}

	// Create game
	g, err := c.gameController.CreateGame(ctx, code, playerIDs, lobby.Config.GridSize)
	if err != nil {
		return nil, err
	}

	// Update lobby state
	lobby.State = model.LobbyStateInGame
	lobby.CurrentGame = &g.ID
	lobby.UpdatedAt = c.clock.Now()

	if err := c.storage.SaveLobby(ctx, lobby); err != nil {
		return nil, err
	}

	return g, nil
}

// AbandonGame ends the current game
func (c *Controller) AbandonGame(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Verify requester is host
	host := lobby.GetHost()
	if host == nil || host.Player.ID != requestingPlayer {
		return model.ErrNotHost
	}

	// Must have game in progress
	if lobby.State != model.LobbyStateInGame || lobby.CurrentGame == nil {
		return model.ErrNoGameInProgress
	}

	// Abandon the game
	if err := c.gameController.AbandonGame(ctx, *lobby.CurrentGame); err != nil {
		return err
	}

	// Update lobby state
	lobby.State = model.LobbyStateWaiting
	lobby.CurrentGame = nil
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// CompleteGame handles a game completing (called when game reaches scoring state)
func (c *Controller) CompleteGame(ctx context.Context, code model.LobbyCode) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	if lobby.CurrentGame == nil {
		return model.ErrNoGameInProgress
	}

	// Create game summary
	summary, err := c.gameController.CreateGameSummary(ctx, *lobby.CurrentGame)
	if err != nil {
		return err
	}

	// Add to history
	lobby.GameHistory = append(lobby.GameHistory, *summary)
	lobby.State = model.LobbyStateWaiting
	lobby.CurrentGame = nil
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// UpdateConfig updates the lobby configuration
func (c *Controller) UpdateConfig(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID, config model.LobbyConfig) error {
	lobby, err := c.storage.GetLobby(ctx, code)
	if err != nil {
		return err
	}

	// Verify requester is host
	host := lobby.GetHost()
	if host == nil || host.Player.ID != requestingPlayer {
		return model.ErrNotHost
	}

	// Cannot change config during game
	if lobby.State == model.LobbyStateInGame {
		return model.ErrGameInProgress
	}

	lobby.Config = config
	lobby.UpdatedAt = c.clock.Now()

	return c.storage.SaveLobby(ctx, lobby)
}

// Interface for dependency injection
type ControllerInterface interface {
	CreateLobby(ctx context.Context, host model.Player) (*model.Lobby, error)
	GetLobby(ctx context.Context, code model.LobbyCode) (*model.Lobby, error)
	JoinLobby(ctx context.Context, code model.LobbyCode, player model.Player) error
	LeaveLobby(ctx context.Context, code model.LobbyCode, playerID model.PlayerID) error
	SetRole(ctx context.Context, code model.LobbyCode, playerID model.PlayerID, role model.LobbyMemberRole) error
	TransferHost(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID, newHostID model.PlayerID) error
	StartGame(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID) (*model.Game, error)
	AbandonGame(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID) error
	CompleteGame(ctx context.Context, code model.LobbyCode) error
	UpdateConfig(ctx context.Context, code model.LobbyCode, requestingPlayer model.PlayerID, config model.LobbyConfig) error
}

var _ ControllerInterface = (*Controller)(nil)
