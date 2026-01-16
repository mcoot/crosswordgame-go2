package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

type StorageSuite struct {
	suite.Suite
	mini    *miniredis.Miniredis
	storage *Storage
	ctx     context.Context
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupTest() {
	s.mini = miniredis.RunT(s.T())

	client := redis.NewClient(&redis.Options{
		Addr: s.mini.Addr(),
	})

	cfg := DefaultConfig()
	cfg.GuestPlayerTTL = time.Hour
	cfg.LobbyTTL = time.Hour
	cfg.GameTTL = time.Hour
	cfg.BoardTTL = time.Hour

	s.storage = NewWithClient(client, cfg)
	s.ctx = context.Background()
}

func (s *StorageSuite) TearDownTest() {
	if s.storage != nil {
		_ = s.storage.Close()
	}
	if s.mini != nil {
		s.mini.Close()
	}
}

// Player tests

func (s *StorageSuite) TestSaveAndGetPlayer() {
	player := &model.Player{
		ID:          "player-1",
		DisplayName: "Alice",
		IsGuest:     false,
		CreatedAt:   time.Now(),
	}

	err := s.storage.SavePlayer(s.ctx, player)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetPlayer(s.ctx, "player-1")
	s.Require().NoError(err)
	s.Equal(player.ID, retrieved.ID)
	s.Equal(player.DisplayName, retrieved.DisplayName)
}

func (s *StorageSuite) TestGetPlayerNotFound() {
	_, err := s.storage.GetPlayer(s.ctx, "nonexistent")
	s.ErrorIs(err, model.ErrPlayerNotFound)
}

func (s *StorageSuite) TestDeletePlayer() {
	player := &model.Player{ID: "player-1", DisplayName: "Alice"}
	_ = s.storage.SavePlayer(s.ctx, player)

	err := s.storage.DeletePlayer(s.ctx, "player-1")
	s.Require().NoError(err)

	_, err = s.storage.GetPlayer(s.ctx, "player-1")
	s.ErrorIs(err, model.ErrPlayerNotFound)
}

func (s *StorageSuite) TestGuestPlayerTTL() {
	guestPlayer := &model.Player{
		ID:      "guest-1",
		IsGuest: true,
	}
	registeredPlayer := &model.Player{
		ID:      "registered-1",
		IsGuest: false,
	}

	_ = s.storage.SavePlayer(s.ctx, guestPlayer)
	_ = s.storage.SavePlayer(s.ctx, registeredPlayer)

	// Check that guest has TTL and registered doesn't
	guestTTL := s.mini.TTL(playerKey(guestPlayer.ID))
	registeredTTL := s.mini.TTL(playerKey(registeredPlayer.ID))

	s.True(guestTTL > 0, "Guest player should have TTL")
	s.Equal(time.Duration(0), registeredTTL, "Registered player should not have TTL")
}

// Registered player tests

func (s *StorageSuite) TestSaveAndGetRegisteredPlayer() {
	rp := &model.RegisteredPlayer{
		PlayerID:     "player-1",
		Username:     "alice",
		PasswordHash: "hash123",
		CreatedAt:    time.Now(),
	}

	err := s.storage.SaveRegisteredPlayer(s.ctx, rp)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetRegisteredPlayer(s.ctx, "player-1")
	s.Require().NoError(err)
	s.Equal(rp.Username, retrieved.Username)
}

func (s *StorageSuite) TestGetRegisteredPlayerByUsername() {
	rp := &model.RegisteredPlayer{
		PlayerID:     "player-1",
		Username:     "alice",
		PasswordHash: "hash123",
	}
	_ = s.storage.SaveRegisteredPlayer(s.ctx, rp)

	retrieved, err := s.storage.GetRegisteredPlayerByUsername(s.ctx, "alice")
	s.Require().NoError(err)
	s.Equal("player-1", string(retrieved.PlayerID))
}

func (s *StorageSuite) TestGetRegisteredPlayerByUsernameNotFound() {
	_, err := s.storage.GetRegisteredPlayerByUsername(s.ctx, "nonexistent")
	s.ErrorIs(err, model.ErrPlayerNotFound)
}

// Lobby tests

func (s *StorageSuite) TestSaveAndGetLobby() {
	lobby := &model.Lobby{
		Code:      "ABC123",
		State:     model.LobbyStateWaiting,
		Config:    model.DefaultLobbyConfig(),
		CreatedAt: time.Now(),
	}

	err := s.storage.SaveLobby(s.ctx, lobby)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetLobby(s.ctx, "ABC123")
	s.Require().NoError(err)
	s.Equal(lobby.Code, retrieved.Code)
	s.Equal(lobby.State, retrieved.State)
}

func (s *StorageSuite) TestGetLobbyNotFound() {
	_, err := s.storage.GetLobby(s.ctx, "NONEXISTENT")
	s.ErrorIs(err, model.ErrLobbyNotFound)
}

func (s *StorageSuite) TestLobbyExists() {
	lobby := &model.Lobby{Code: "ABC123", State: model.LobbyStateWaiting}
	_ = s.storage.SaveLobby(s.ctx, lobby)

	exists, err := s.storage.LobbyExists(s.ctx, "ABC123")
	s.Require().NoError(err)
	s.True(exists)

	exists, err = s.storage.LobbyExists(s.ctx, "NONEXISTENT")
	s.Require().NoError(err)
	s.False(exists)
}

func (s *StorageSuite) TestDeleteLobby() {
	lobby := &model.Lobby{Code: "ABC123", State: model.LobbyStateWaiting}
	_ = s.storage.SaveLobby(s.ctx, lobby)

	err := s.storage.DeleteLobby(s.ctx, "ABC123")
	s.Require().NoError(err)

	_, err = s.storage.GetLobby(s.ctx, "ABC123")
	s.ErrorIs(err, model.ErrLobbyNotFound)
}

func (s *StorageSuite) TestLobbyTTL() {
	lobby := &model.Lobby{Code: "ABC123", State: model.LobbyStateWaiting}
	_ = s.storage.SaveLobby(s.ctx, lobby)

	ttl := s.mini.TTL(lobbyKey(lobby.Code))
	s.True(ttl > 0, "Lobby should have TTL")
}

// Game tests

func (s *StorageSuite) TestSaveAndGetGame() {
	game := &model.Game{
		ID:        "game-1",
		LobbyCode: "ABC123",
		State:     model.GameStateAnnouncing,
		GridSize:  5,
		Players:   []model.PlayerID{"p1", "p2"},
	}

	err := s.storage.SaveGame(s.ctx, game)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetGame(s.ctx, "game-1")
	s.Require().NoError(err)
	s.Equal(game.ID, retrieved.ID)
	s.Equal(game.State, retrieved.State)
}

func (s *StorageSuite) TestGetGameNotFound() {
	_, err := s.storage.GetGame(s.ctx, "nonexistent")
	s.ErrorIs(err, model.ErrGameNotFound)
}

func (s *StorageSuite) TestGameTTL() {
	game := &model.Game{ID: "game-1", State: model.GameStateAnnouncing}
	_ = s.storage.SaveGame(s.ctx, game)

	ttl := s.mini.TTL(gameKey(game.ID))
	s.True(ttl > 0, "Game should have TTL")
}

// Board tests

func (s *StorageSuite) TestSaveAndGetBoard() {
	board := model.NewBoard("game-1", "player-1", 5)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')

	err := s.storage.SaveBoard(s.ctx, board)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetBoard(s.ctx, "game-1", "player-1")
	s.Require().NoError(err)
	s.Equal(board.Size, retrieved.Size)
	s.Equal('A', retrieved.Get(model.Position{Row: 0, Col: 0}))
}

func (s *StorageSuite) TestGetBoardNotFound() {
	_, err := s.storage.GetBoard(s.ctx, "game-1", "nonexistent")
	s.ErrorIs(err, model.ErrBoardNotFound)
}

func (s *StorageSuite) TestGetBoardsForGame() {
	board1 := model.NewBoard("game-1", "player-1", 5)
	board2 := model.NewBoard("game-1", "player-2", 5)
	board3 := model.NewBoard("game-2", "player-1", 5) // Different game

	_ = s.storage.SaveBoard(s.ctx, board1)
	_ = s.storage.SaveBoard(s.ctx, board2)
	_ = s.storage.SaveBoard(s.ctx, board3)

	boards, err := s.storage.GetBoardsForGame(s.ctx, "game-1")
	s.Require().NoError(err)
	s.Len(boards, 2)
}

func (s *StorageSuite) TestGetBoardsForGameEmpty() {
	boards, err := s.storage.GetBoardsForGame(s.ctx, "nonexistent")
	s.Require().NoError(err)
	s.Empty(boards)
}

func (s *StorageSuite) TestDeleteBoardsForGame() {
	board1 := model.NewBoard("game-1", "player-1", 5)
	board2 := model.NewBoard("game-1", "player-2", 5)
	_ = s.storage.SaveBoard(s.ctx, board1)
	_ = s.storage.SaveBoard(s.ctx, board2)

	err := s.storage.DeleteBoardsForGame(s.ctx, "game-1")
	s.Require().NoError(err)

	boards, err := s.storage.GetBoardsForGame(s.ctx, "game-1")
	s.Require().NoError(err)
	s.Empty(boards)
}

func (s *StorageSuite) TestBoardTTL() {
	board := model.NewBoard("game-1", "player-1", 5)
	_ = s.storage.SaveBoard(s.ctx, board)

	ttl := s.mini.TTL(boardKey(board.GameID, board.PlayerID))
	s.True(ttl > 0, "Board should have TTL")
}

// Dictionary tests

func (s *StorageSuite) TestSaveAndGetDictionaryWords() {
	words := []string{"apple", "banana", "cherry"}

	err := s.storage.SaveDictionaryWords(s.ctx, words)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetDictionaryWords(s.ctx)
	s.Require().NoError(err)
	s.ElementsMatch(words, retrieved) // Order may differ (SET)
}

func (s *StorageSuite) TestGetDictionaryWordsNotLoaded() {
	_, err := s.storage.GetDictionaryWords(s.ctx)
	s.ErrorIs(err, model.ErrDictionaryNotLoaded)
}

func (s *StorageSuite) TestSaveDictionaryWordsReplacesExisting() {
	words1 := []string{"apple", "banana"}
	words2 := []string{"cherry", "date", "elderberry"}

	_ = s.storage.SaveDictionaryWords(s.ctx, words1)
	_ = s.storage.SaveDictionaryWords(s.ctx, words2)

	retrieved, err := s.storage.GetDictionaryWords(s.ctx)
	s.Require().NoError(err)
	s.ElementsMatch(words2, retrieved)
}

func (s *StorageSuite) TestDictionaryNoTTL() {
	words := []string{"apple"}
	_ = s.storage.SaveDictionaryWords(s.ctx, words)

	ttl := s.mini.TTL(dictionaryKey())
	s.Equal(time.Duration(0), ttl, "Dictionary should not have TTL")
}
