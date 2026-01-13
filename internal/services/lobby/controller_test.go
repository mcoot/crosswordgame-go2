package lobby

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/mcoot/crosswordgame-go2/internal/testutil"
)

type ControllerSuite struct {
	suite.Suite
	storage        *memory.Storage
	gameController *game.Controller
	clock          *mocks.MockClock
	random         *mocks.MockRandom
	controller     *Controller
	ctx            context.Context
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerSuite))
}

func (s *ControllerSuite) SetupTest() {
	s.storage = memory.New()
	logger := testutil.NopLogger()
	boardService := board.New(s.storage, logger)
	dictService := dictionary.New(s.storage, logger)
	scoringService := scoring.New(dictService)
	s.clock = mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	s.random = mocks.NewMockRandom()
	s.gameController = game.NewController(s.storage, boardService, scoringService, s.clock, s.random, logger)
	s.controller = NewController(s.storage, s.gameController, s.clock, s.random, logger)
	s.ctx = context.Background()

	// Load dictionary
	_ = dictService.LoadWords([]string{"cat", "dog", "at", "to"})
}

func (s *ControllerSuite) createPlayer(id string, name string) model.Player {
	return model.Player{
		ID:          model.PlayerID(id),
		DisplayName: name,
		IsGuest:     true,
		CreatedAt:   s.clock.Now(),
	}
}

// CreateLobby tests

func (s *ControllerSuite) TestCreateLobbySucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")

	lobby, err := s.controller.CreateLobby(s.ctx, host)
	s.Require().NoError(err)

	s.Equal(model.LobbyCode("ABC123"), lobby.Code)
	s.Equal(model.LobbyStateWaiting, lobby.State)
	s.Len(lobby.Members, 1)
	s.Equal(host.ID, lobby.Members[0].Player.ID)
	s.True(lobby.Members[0].IsHost)
	s.Equal(model.RolePlayer, lobby.Members[0].Role)
}

func (s *ControllerSuite) TestCreateLobbyIsPersisted() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")

	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	retrieved, err := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Require().NoError(err)
	s.Equal(lobby.Code, retrieved.Code)
}

func (s *ControllerSuite) TestCreateLobbyHasDefaultConfig() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")

	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	s.Equal(5, lobby.Config.GridSize)
}

// JoinLobby tests

func (s *ControllerSuite) TestJoinLobbySucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	err := s.controller.JoinLobby(s.ctx, lobby.Code, player)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Len(updated.Members, 2)
	s.Equal(model.RolePlayer, updated.GetMember(player.ID).Role)
}

func (s *ControllerSuite) TestJoinLobbyDuringGameAsSpectator() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	// Start a game
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	// New player joins as spectator
	player := s.createPlayer("player-1", "Player")
	err := s.controller.JoinLobby(s.ctx, lobby.Code, player)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.RoleSpectator, updated.GetMember(player.ID).Role)
}

func (s *ControllerSuite) TestJoinLobbyFailsIfAlreadyMember() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.JoinLobby(s.ctx, lobby.Code, host)
	s.ErrorIs(err, model.ErrAlreadyInLobby)
}

func (s *ControllerSuite) TestJoinLobbyFailsIfNotFound() {
	player := s.createPlayer("player-1", "Player")
	err := s.controller.JoinLobby(s.ctx, "NONEXISTENT", player)
	s.ErrorIs(err, model.ErrLobbyNotFound)
}

// LeaveLobby tests

func (s *ControllerSuite) TestLeaveLobbySucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	err := s.controller.LeaveLobby(s.ctx, lobby.Code, player.ID)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Len(updated.Members, 1)
	s.Nil(updated.GetMember(player.ID))
}

func (s *ControllerSuite) TestLeaveLobbyDeletesEmptyLobby() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.LeaveLobby(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)

	_, err = s.controller.GetLobby(s.ctx, lobby.Code)
	s.ErrorIs(err, model.ErrLobbyNotFound)
}

func (s *ControllerSuite) TestLeaveLobbyTransfersHost() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	// Host leaves
	err := s.controller.LeaveLobby(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.True(updated.Members[0].IsHost)
	s.Equal(player.ID, updated.Members[0].Player.ID)
}

func (s *ControllerSuite) TestLeaveLobbyFailsIfNotMember() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.LeaveLobby(s.ctx, lobby.Code, "nonexistent")
	s.ErrorIs(err, model.ErrNotInLobby)
}

// SetRole tests

func (s *ControllerSuite) TestSetRoleSucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	err := s.controller.SetRole(s.ctx, lobby.Code, player.ID, model.RoleSpectator)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.RoleSpectator, updated.GetMember(player.ID).Role)
}

func (s *ControllerSuite) TestSetRoleFailsDuringGame() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	err := s.controller.SetRole(s.ctx, lobby.Code, host.ID, model.RoleSpectator)
	s.ErrorIs(err, model.ErrGameInProgress)
}

func (s *ControllerSuite) TestSetRoleFailsIfNotMember() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.SetRole(s.ctx, lobby.Code, "nonexistent", model.RoleSpectator)
	s.ErrorIs(err, model.ErrNotInLobby)
}

// TransferHost tests

func (s *ControllerSuite) TestTransferHostSucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	err := s.controller.TransferHost(s.ctx, lobby.Code, host.ID, player.ID)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.True(updated.GetMember(player.ID).IsHost)
	s.False(updated.GetMember(host.ID).IsHost)
}

func (s *ControllerSuite) TestTransferHostFailsIfNotHost() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	// Non-host tries to transfer
	err := s.controller.TransferHost(s.ctx, lobby.Code, player.ID, host.ID)
	s.ErrorIs(err, model.ErrNotHost)
}

func (s *ControllerSuite) TestTransferHostFailsIfTargetNotInLobby() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.TransferHost(s.ctx, lobby.Code, host.ID, "nonexistent")
	s.ErrorIs(err, model.ErrNotInLobby)
}

// StartGame tests

func (s *ControllerSuite) TestStartGameSucceeds() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	game, err := s.controller.StartGame(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)
	s.NotNil(game)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.LobbyStateInGame, updated.State)
	s.Equal(&game.ID, updated.CurrentGame)
}

func (s *ControllerSuite) TestStartGameFailsIfNotHost() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	_, err := s.controller.StartGame(s.ctx, lobby.Code, player.ID)
	s.ErrorIs(err, model.ErrNotHost)
}

func (s *ControllerSuite) TestStartGameFailsIfGameInProgress() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	_, err := s.controller.StartGame(s.ctx, lobby.Code, host.ID)
	s.ErrorIs(err, model.ErrGameInProgress)
}

func (s *ControllerSuite) TestStartGameFailsIfNoPlayers() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	// Make host a spectator
	_ = s.controller.SetRole(s.ctx, lobby.Code, host.ID, model.RoleSpectator)

	_, err := s.controller.StartGame(s.ctx, lobby.Code, host.ID)
	s.ErrorIs(err, model.ErrInsufficientPlayers)
}

func (s *ControllerSuite) TestStartGameOnlyIncludesPlayers() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	spectator := s.createPlayer("spectator-1", "Spectator")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, spectator)
	_ = s.controller.SetRole(s.ctx, lobby.Code, spectator.ID, model.RoleSpectator)

	game, _ := s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	s.Len(game.Players, 2)
	s.Contains(game.Players, host.ID)
	s.Contains(game.Players, player.ID)
	s.NotContains(game.Players, spectator.ID)
}

// AbandonGame tests

func (s *ControllerSuite) TestAbandonGameSucceeds() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	err := s.controller.AbandonGame(s.ctx, lobby.Code, host.ID)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.LobbyStateWaiting, updated.State)
	s.Nil(updated.CurrentGame)
}

func (s *ControllerSuite) TestAbandonGameFailsIfNotHost() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	err := s.controller.AbandonGame(s.ctx, lobby.Code, player.ID)
	s.ErrorIs(err, model.ErrNotHost)
}

func (s *ControllerSuite) TestAbandonGameFailsIfNoGame() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	err := s.controller.AbandonGame(s.ctx, lobby.Code, host.ID)
	s.ErrorIs(err, model.ErrNoGameInProgress)
}

// UpdateConfig tests

func (s *ControllerSuite) TestUpdateConfigSucceeds() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	newConfig := model.LobbyConfig{GridSize: 7}
	err := s.controller.UpdateConfig(s.ctx, lobby.Code, host.ID, newConfig)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(7, updated.Config.GridSize)
}

func (s *ControllerSuite) TestUpdateConfigFailsIfNotHost() {
	s.random.QueueString("ABC123")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	player := s.createPlayer("player-1", "Player")
	_ = s.controller.JoinLobby(s.ctx, lobby.Code, player)

	err := s.controller.UpdateConfig(s.ctx, lobby.Code, player.ID, model.LobbyConfig{GridSize: 7})
	s.ErrorIs(err, model.ErrNotHost)
}

func (s *ControllerSuite) TestUpdateConfigFailsDuringGame() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)
	_, _ = s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	err := s.controller.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 7})
	s.ErrorIs(err, model.ErrGameInProgress)
}

// CompleteGame tests

func (s *ControllerSuite) TestCompleteGameAddsToHistory() {
	s.random.QueueString("ABC123", "GAME12345678")
	host := s.createPlayer("host-1", "Host")
	lobby, _ := s.controller.CreateLobby(s.ctx, host)

	// Use a 2x2 grid for quick completion
	_ = s.controller.UpdateConfig(s.ctx, lobby.Code, host.ID, model.LobbyConfig{GridSize: 2})
	g, _ := s.controller.StartGame(s.ctx, lobby.Code, host.ID)

	// Complete the game
	positions := []model.Position{{Row: 0, Col: 0}, {Row: 0, Col: 1}, {Row: 1, Col: 0}, {Row: 1, Col: 1}}
	for i, pos := range positions {
		_ = s.gameController.AnnounceLetter(s.ctx, g.ID, host.ID, rune('A'+i))
		_ = s.gameController.PlaceLetter(s.ctx, g.ID, host.ID, pos)
	}

	err := s.controller.CompleteGame(s.ctx, lobby.Code)
	s.Require().NoError(err)

	updated, _ := s.controller.GetLobby(s.ctx, lobby.Code)
	s.Equal(model.LobbyStateWaiting, updated.State)
	s.Nil(updated.CurrentGame)
	s.Len(updated.GameHistory, 1)
	s.Equal(g.ID, updated.GameHistory[0].ID)
}
