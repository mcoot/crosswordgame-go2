package storage

import (
	"context"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// Storage defines the interface for data persistence
type Storage interface {
	// Player operations
	SavePlayer(ctx context.Context, player *model.Player) error
	GetPlayer(ctx context.Context, id model.PlayerID) (*model.Player, error)
	DeletePlayer(ctx context.Context, id model.PlayerID) error

	// Registered player operations
	SaveRegisteredPlayer(ctx context.Context, rp *model.RegisteredPlayer) error
	GetRegisteredPlayer(ctx context.Context, playerID model.PlayerID) (*model.RegisteredPlayer, error)
	GetRegisteredPlayerByUsername(ctx context.Context, username string) (*model.RegisteredPlayer, error)

	// Lobby operations
	SaveLobby(ctx context.Context, lobby *model.Lobby) error
	GetLobby(ctx context.Context, code model.LobbyCode) (*model.Lobby, error)
	DeleteLobby(ctx context.Context, code model.LobbyCode) error
	LobbyExists(ctx context.Context, code model.LobbyCode) (bool, error)

	// Game operations
	SaveGame(ctx context.Context, game *model.Game) error
	GetGame(ctx context.Context, id model.GameID) (*model.Game, error)
	DeleteGame(ctx context.Context, id model.GameID) error

	// Board operations
	SaveBoard(ctx context.Context, board *model.Board) error
	GetBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID) (*model.Board, error)
	GetBoardsForGame(ctx context.Context, gameID model.GameID) ([]*model.Board, error)
	DeleteBoardsForGame(ctx context.Context, gameID model.GameID) error

	// Dictionary operations
	GetDictionaryWords(ctx context.Context) ([]string, error)
	SaveDictionaryWords(ctx context.Context, words []string) error
}
