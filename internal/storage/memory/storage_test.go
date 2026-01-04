package memory

import (
	"context"
	"testing"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/stretchr/testify/suite"
)

type StorageSuite struct {
	suite.Suite
	storage *Storage
	ctx     context.Context
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupTest() {
	s.storage = New()
	s.ctx = context.Background()
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

// Dictionary tests

func (s *StorageSuite) TestSaveAndGetDictionaryWords() {
	words := []string{"apple", "banana", "cherry"}

	err := s.storage.SaveDictionaryWords(s.ctx, words)
	s.Require().NoError(err)

	retrieved, err := s.storage.GetDictionaryWords(s.ctx)
	s.Require().NoError(err)
	s.Equal(words, retrieved)
}

func (s *StorageSuite) TestGetDictionaryWordsNotLoaded() {
	_, err := s.storage.GetDictionaryWords(s.ctx)
	s.ErrorIs(err, model.ErrDictionaryNotLoaded)
}
