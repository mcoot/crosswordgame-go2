package memory

import (
	"context"
	"sync"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Storage is an in-memory implementation of the storage interface
type Storage struct {
	mu sync.RWMutex

	players           map[model.PlayerID]*model.Player
	registeredPlayers map[model.PlayerID]*model.RegisteredPlayer
	usernameIndex     map[string]model.PlayerID
	lobbies           map[model.LobbyCode]*model.Lobby
	games             map[model.GameID]*model.Game
	boards            map[boardKey]*model.Board
	dictionaryWords   []string
}

type boardKey struct {
	gameID   model.GameID
	playerID model.PlayerID
}

// New creates a new in-memory storage instance
func New() *Storage {
	return &Storage{
		players:           make(map[model.PlayerID]*model.Player),
		registeredPlayers: make(map[model.PlayerID]*model.RegisteredPlayer),
		usernameIndex:     make(map[string]model.PlayerID),
		lobbies:           make(map[model.LobbyCode]*model.Lobby),
		games:             make(map[model.GameID]*model.Game),
		boards:            make(map[boardKey]*model.Board),
	}
}

// Ensure Storage implements the interface
var _ storage.Storage = (*Storage)(nil)

// Player operations

func (s *Storage) SavePlayer(ctx context.Context, player *model.Player) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.players[player.ID] = player
	return nil
}

func (s *Storage) GetPlayer(ctx context.Context, id model.PlayerID) (*model.Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	player, ok := s.players[id]
	if !ok {
		return nil, model.ErrPlayerNotFound
	}
	return player, nil
}

func (s *Storage) DeletePlayer(ctx context.Context, id model.PlayerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.players, id)
	return nil
}

// Registered player operations

func (s *Storage) SaveRegisteredPlayer(ctx context.Context, rp *model.RegisteredPlayer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registeredPlayers[rp.PlayerID] = rp
	s.usernameIndex[rp.Username] = rp.PlayerID
	return nil
}

func (s *Storage) GetRegisteredPlayer(ctx context.Context, playerID model.PlayerID) (*model.RegisteredPlayer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rp, ok := s.registeredPlayers[playerID]
	if !ok {
		return nil, model.ErrPlayerNotFound
	}
	return rp, nil
}

func (s *Storage) GetRegisteredPlayerByUsername(ctx context.Context, username string) (*model.RegisteredPlayer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	playerID, ok := s.usernameIndex[username]
	if !ok {
		return nil, model.ErrPlayerNotFound
	}
	rp, ok := s.registeredPlayers[playerID]
	if !ok {
		return nil, model.ErrPlayerNotFound
	}
	return rp, nil
}

// Lobby operations

func (s *Storage) SaveLobby(ctx context.Context, lobby *model.Lobby) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lobbies[lobby.Code] = lobby
	return nil
}

func (s *Storage) GetLobby(ctx context.Context, code model.LobbyCode) (*model.Lobby, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lobby, ok := s.lobbies[code]
	if !ok {
		return nil, model.ErrLobbyNotFound
	}
	return lobby, nil
}

func (s *Storage) DeleteLobby(ctx context.Context, code model.LobbyCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.lobbies, code)
	return nil
}

func (s *Storage) LobbyExists(ctx context.Context, code model.LobbyCode) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.lobbies[code]
	return ok, nil
}

// Game operations

func (s *Storage) SaveGame(ctx context.Context, game *model.Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[game.ID] = game
	return nil
}

func (s *Storage) GetGame(ctx context.Context, id model.GameID) (*model.Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	game, ok := s.games[id]
	if !ok {
		return nil, model.ErrGameNotFound
	}
	return game, nil
}

func (s *Storage) DeleteGame(ctx context.Context, id model.GameID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, id)
	return nil
}

// Board operations

func (s *Storage) SaveBoard(ctx context.Context, board *model.Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := boardKey{gameID: board.GameID, playerID: board.PlayerID}
	s.boards[key] = board
	return nil
}

func (s *Storage) GetBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID) (*model.Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := boardKey{gameID: gameID, playerID: playerID}
	board, ok := s.boards[key]
	if !ok {
		return nil, model.ErrBoardNotFound
	}
	return board, nil
}

func (s *Storage) GetBoardsForGame(ctx context.Context, gameID model.GameID) ([]*model.Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var boards []*model.Board
	for key, board := range s.boards {
		if key.gameID == gameID {
			boards = append(boards, board)
		}
	}
	return boards, nil
}

func (s *Storage) DeleteBoardsForGame(ctx context.Context, gameID model.GameID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.boards {
		if key.gameID == gameID {
			delete(s.boards, key)
		}
	}
	return nil
}

// Dictionary operations

func (s *Storage) GetDictionaryWords(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.dictionaryWords == nil {
		return nil, model.ErrDictionaryNotLoaded
	}
	result := make([]string, len(s.dictionaryWords))
	copy(result, s.dictionaryWords)
	return result, nil
}

func (s *Storage) SaveDictionaryWords(ctx context.Context, words []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dictionaryWords = make([]string, len(words))
	copy(s.dictionaryWords, words)
	return nil
}
