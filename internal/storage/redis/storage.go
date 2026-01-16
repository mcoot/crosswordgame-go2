package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Storage is a Redis-backed implementation of the storage interface
type Storage struct {
	client *redis.Client
	cfg    Config
}

// New creates a new Redis storage instance
func New(cfg Config) (*Storage, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Storage{
		client: client,
		cfg:    cfg,
	}, nil
}

// NewWithClient creates a Redis storage with an existing client (for testing)
func NewWithClient(client *redis.Client, cfg Config) *Storage {
	return &Storage{
		client: client,
		cfg:    cfg,
	}
}

// Close closes the Redis connection
func (s *Storage) Close() error {
	return s.client.Close()
}

// Ensure Storage implements the interface
var _ storage.Storage = (*Storage)(nil)

// Player operations

func (s *Storage) SavePlayer(ctx context.Context, player *model.Player) error {
	data, err := json.Marshal(player)
	if err != nil {
		return err
	}

	key := playerKey(player.ID)

	// Apply TTL only for guest players
	var ttl time.Duration
	if player.IsGuest {
		ttl = s.cfg.GuestPlayerTTL
	}

	if ttl > 0 {
		return s.client.Set(ctx, key, data, ttl).Err()
	}
	return s.client.Set(ctx, key, data, 0).Err()
}

func (s *Storage) GetPlayer(ctx context.Context, id model.PlayerID) (*model.Player, error) {
	data, err := s.client.Get(ctx, playerKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrPlayerNotFound
		}
		return nil, err
	}

	var player model.Player
	if err := json.Unmarshal(data, &player); err != nil {
		return nil, err
	}
	return &player, nil
}

func (s *Storage) DeletePlayer(ctx context.Context, id model.PlayerID) error {
	return s.client.Del(ctx, playerKey(id)).Err()
}

// Registered player operations

func (s *Storage) SaveRegisteredPlayer(ctx context.Context, rp *model.RegisteredPlayer) error {
	data, err := json.Marshal(rp)
	if err != nil {
		return err
	}

	// Use pipeline for atomic save + index update
	pipe := s.client.Pipeline()
	pipe.Set(ctx, registeredPlayerKey(rp.PlayerID), data, 0) // No TTL
	pipe.Set(ctx, usernameIndexKey(rp.Username), string(rp.PlayerID), 0)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Storage) GetRegisteredPlayer(ctx context.Context, playerID model.PlayerID) (*model.RegisteredPlayer, error) {
	data, err := s.client.Get(ctx, registeredPlayerKey(playerID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrPlayerNotFound
		}
		return nil, err
	}

	var rp model.RegisteredPlayer
	if err := json.Unmarshal(data, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}

func (s *Storage) GetRegisteredPlayerByUsername(ctx context.Context, username string) (*model.RegisteredPlayer, error) {
	// Look up player ID from username index
	playerIDStr, err := s.client.Get(ctx, usernameIndexKey(username)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrPlayerNotFound
		}
		return nil, err
	}

	return s.GetRegisteredPlayer(ctx, model.PlayerID(playerIDStr))
}

// Lobby operations

func (s *Storage) SaveLobby(ctx context.Context, lobby *model.Lobby) error {
	data, err := json.Marshal(lobby)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, lobbyKey(lobby.Code), data, s.cfg.LobbyTTL).Err()
}

func (s *Storage) GetLobby(ctx context.Context, code model.LobbyCode) (*model.Lobby, error) {
	data, err := s.client.Get(ctx, lobbyKey(code)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrLobbyNotFound
		}
		return nil, err
	}

	var lobby model.Lobby
	if err := json.Unmarshal(data, &lobby); err != nil {
		return nil, err
	}
	return &lobby, nil
}

func (s *Storage) DeleteLobby(ctx context.Context, code model.LobbyCode) error {
	return s.client.Del(ctx, lobbyKey(code)).Err()
}

func (s *Storage) LobbyExists(ctx context.Context, code model.LobbyCode) (bool, error) {
	exists, err := s.client.Exists(ctx, lobbyKey(code)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Game operations

func (s *Storage) SaveGame(ctx context.Context, game *model.Game) error {
	data, err := json.Marshal(game)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, gameKey(game.ID), data, s.cfg.GameTTL).Err()
}

func (s *Storage) GetGame(ctx context.Context, id model.GameID) (*model.Game, error) {
	data, err := s.client.Get(ctx, gameKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrGameNotFound
		}
		return nil, err
	}

	var game model.Game
	if err := json.Unmarshal(data, &game); err != nil {
		return nil, err
	}
	return &game, nil
}

func (s *Storage) DeleteGame(ctx context.Context, id model.GameID) error {
	return s.client.Del(ctx, gameKey(id)).Err()
}

// Board operations

func (s *Storage) SaveBoard(ctx context.Context, board *model.Board) error {
	data, err := json.Marshal(board)
	if err != nil {
		return err
	}

	bKey := boardKey(board.GameID, board.PlayerID)
	indexKey := boardsForGameIndexKey(board.GameID)

	// Use pipeline for atomic save + index update
	pipe := s.client.Pipeline()
	pipe.Set(ctx, bKey, data, s.cfg.BoardTTL)
	pipe.SAdd(ctx, indexKey, bKey)
	pipe.Expire(ctx, indexKey, s.cfg.BoardTTL) // Keep index TTL in sync
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Storage) GetBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID) (*model.Board, error) {
	data, err := s.client.Get(ctx, boardKey(gameID, playerID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrBoardNotFound
		}
		return nil, err
	}

	var board model.Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, err
	}
	return &board, nil
}

func (s *Storage) GetBoardsForGame(ctx context.Context, gameID model.GameID) ([]*model.Board, error) {
	indexKey := boardsForGameIndexKey(gameID)

	// Get all board keys from the index
	boardKeys, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, err
	}

	if len(boardKeys) == 0 {
		return []*model.Board{}, nil
	}

	// Fetch all boards in parallel using MGET
	values, err := s.client.MGet(ctx, boardKeys...).Result()
	if err != nil {
		return nil, err
	}

	boards := make([]*model.Board, 0, len(values))
	for _, val := range values {
		if val == nil {
			continue // Board may have expired
		}
		var board model.Board
		if err := json.Unmarshal([]byte(val.(string)), &board); err != nil {
			continue // Skip invalid data
		}
		boards = append(boards, &board)
	}

	return boards, nil
}

func (s *Storage) DeleteBoardsForGame(ctx context.Context, gameID model.GameID) error {
	indexKey := boardsForGameIndexKey(gameID)

	// Get all board keys from the index
	boardKeys, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return err
	}

	if len(boardKeys) == 0 {
		return nil
	}

	// Delete all boards and the index in one pipeline
	pipe := s.client.Pipeline()
	for _, key := range boardKeys {
		pipe.Del(ctx, key)
	}
	pipe.Del(ctx, indexKey)
	_, err = pipe.Exec(ctx)
	return err
}

// Dictionary operations

func (s *Storage) GetDictionaryWords(ctx context.Context) ([]string, error) {
	key := dictionaryKey()

	// Check if dictionary exists
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, model.ErrDictionaryNotLoaded
	}

	// Get all words from the set
	words, err := s.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return words, nil
}

func (s *Storage) SaveDictionaryWords(ctx context.Context, words []string) error {
	key := dictionaryKey()

	// Delete existing dictionary and add new words atomically
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)

	if len(words) > 0 {
		// Convert []string to []interface{} for SAdd
		members := make([]interface{}, len(words))
		for i, w := range words {
			members[i] = w
		}
		pipe.SAdd(ctx, key, members...)
	}

	_, err := pipe.Exec(ctx)
	return err
}
