package game

import (
	"context"
	"testing"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/stretchr/testify/suite"
)

type ControllerSuite struct {
	suite.Suite
	storage        *memory.Storage
	boardService   *board.Service
	dictService    *dictionary.Service
	scoringService *scoring.Service
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
	s.boardService = board.New(s.storage)
	s.dictService = dictionary.New(s.storage)
	s.scoringService = scoring.New(s.dictService)
	s.clock = mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	s.random = mocks.NewMockRandom()
	s.controller = NewController(s.storage, s.boardService, s.scoringService, s.clock, s.random)
	s.ctx = context.Background()

	// Load dictionary for scoring tests
	_ = s.dictService.LoadWords([]string{"cat", "dog", "at", "to", "go"})
}

// CreateGame tests

func (s *ControllerSuite) TestCreateGameSucceeds() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}

	game, err := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	s.Require().NoError(err)

	s.Equal(model.GameID("GAME12345678"), game.ID)
	s.Equal(model.LobbyCode("LOBBY1"), game.LobbyCode)
	s.Equal(model.GameStateAnnouncing, game.State)
	s.Equal(5, game.GridSize)
	s.Equal(players, game.Players)
	s.Equal(0, game.CurrentTurn)
	s.Equal(0, game.AnnouncerIdx)
	s.Equal(rune(0), game.CurrentLetter)
}

func (s *ControllerSuite) TestCreateGameCreatesBoardsForAllPlayers() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2", "player-3"}

	game, err := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	s.Require().NoError(err)

	for _, playerID := range players {
		board, err := s.boardService.GetBoard(s.ctx, game.ID, playerID)
		s.Require().NoError(err)
		s.Equal(5, board.Size)
	}
}

func (s *ControllerSuite) TestCreateGameFailsWithNoPlayers() {
	_, err := s.controller.CreateGame(s.ctx, "LOBBY1", []model.PlayerID{}, 5)
	s.ErrorIs(err, model.ErrInsufficientPlayers)
}

func (s *ControllerSuite) TestCreateGameIsPersisted() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}

	game, err := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	s.Require().NoError(err)

	retrieved, err := s.controller.GetGame(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Equal(game.ID, retrieved.ID)
}

// AnnounceLetter tests

func (s *ControllerSuite) TestAnnounceLetterSucceeds() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	s.Require().NoError(err)

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal('A', updated.CurrentLetter)
	s.Equal(model.GameStatePlacing, updated.State)
}

func (s *ControllerSuite) TestAnnounceLetterNormalizesToUppercase() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'a')
	s.Require().NoError(err)

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal('A', updated.CurrentLetter)
}

func (s *ControllerSuite) TestAnnounceLetterFailsIfNotAnnouncer() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	// player-2 is not the announcer (player-1 is first)
	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-2", 'A')
	s.ErrorIs(err, model.ErrNotPlayerTurn)
}

func (s *ControllerSuite) TestAnnounceLetterFailsIfNotAnnouncingState() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	// First announcement succeeds
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')

	// Second announcement fails (now in Placing state)
	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'B')
	s.ErrorIs(err, model.ErrNotPlayerTurn)
}

func (s *ControllerSuite) TestAnnounceLetterFailsWithInvalidLetter() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", '1')
	s.ErrorIs(err, model.ErrInvalidLetter)
}

// PlaceLetter tests

func (s *ControllerSuite) TestPlaceLetterSucceeds() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')

	err := s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})
	s.Require().NoError(err)

	board, _ := s.boardService.GetBoard(s.ctx, game.ID, "player-1")
	s.Equal('A', board.Get(model.Position{Row: 0, Col: 0}))
}

func (s *ControllerSuite) TestPlaceLetterMarksPlayerAsPlaced() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')

	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.True(updated.Placements["player-1"])
	s.False(updated.Placements["player-2"])
}

func (s *ControllerSuite) TestPlaceLetterFailsIfNoLetterAnnounced() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrLetterNotAnnounced)
}

func (s *ControllerSuite) TestPlaceLetterFailsIfAlreadyPlaced() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})

	err := s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 1, Col: 0})
	s.ErrorIs(err, model.ErrAlreadyPlaced)
}

func (s *ControllerSuite) TestPlaceLetterFailsIfCellOccupied() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 2) // 2x2 grid = 4 turns

	// Turn 1
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})

	// Turn 2 - try to place in same spot
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'B')
	err := s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrCellOccupied)
}

func (s *ControllerSuite) TestPlaceLetterFailsForNonPlayer() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')

	err := s.controller.PlaceLetter(s.ctx, game.ID, "player-999", model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrPlayerNotFound)
}

// Turn advancement tests

func (s *ControllerSuite) TestAllPlayersPlacedAdvancesTurn() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	// Turn 1
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-2", model.Position{Row: 0, Col: 0})

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(1, updated.CurrentTurn)
	s.Equal(model.GameStateAnnouncing, updated.State)
	s.Equal(1, updated.AnnouncerIdx) // Rotated to player-2
	s.Equal(rune(0), updated.CurrentLetter)
}

func (s *ControllerSuite) TestAnnouncerRotatesCorrectly() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2", "player-3"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 2) // 2x2 = 4 turns

	// Turn 1: player-1 announces
	s.Equal(model.PlayerID("player-1"), game.CurrentAnnouncer())
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-2", model.Position{Row: 0, Col: 0})
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-3", model.Position{Row: 0, Col: 0})

	// Turn 2: player-2 announces
	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(model.PlayerID("player-2"), updated.CurrentAnnouncer())

	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-2", 'B')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 1})
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-2", model.Position{Row: 0, Col: 1})
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-3", model.Position{Row: 0, Col: 1})

	// Turn 3: player-3 announces
	updated, _ = s.controller.GetGame(s.ctx, game.ID)
	s.Equal(model.PlayerID("player-3"), updated.CurrentAnnouncer())
}

// Game completion tests

func (s *ControllerSuite) TestGameCompletesWhenGridFull() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 2) // 2x2 = 4 turns

	positions := []model.Position{
		{Row: 0, Col: 0}, {Row: 0, Col: 1},
		{Row: 1, Col: 0}, {Row: 1, Col: 1},
	}

	for i, pos := range positions {
		_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", rune('A'+i))
		_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", pos)
	}

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(model.GameStateScoring, updated.State)
	s.Equal(4, updated.CurrentTurn)
}

// AbandonGame tests

func (s *ControllerSuite) TestAbandonGameSucceeds() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.AbandonGame(s.ctx, game.ID)
	s.Require().NoError(err)

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(model.GameStateAbandoned, updated.State)
}

func (s *ControllerSuite) TestAbandonGameIdempotent() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	_ = s.controller.AbandonGame(s.ctx, game.ID)
	err := s.controller.AbandonGame(s.ctx, game.ID)
	s.NoError(err)
}

func (s *ControllerSuite) TestCannotActOnAbandonedGame() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AbandonGame(s.ctx, game.ID)

	err := s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	s.ErrorIs(err, model.ErrGameAbandoned)
}

// RemovePlayer tests

func (s *ControllerSuite) TestRemovePlayerFromGame() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2", "player-3"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.RemovePlayer(s.ctx, game.ID, "player-2")
	s.Require().NoError(err)

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Len(updated.Players, 2)
	s.Contains(updated.Players, model.PlayerID("player-1"))
	s.Contains(updated.Players, model.PlayerID("player-3"))
}

func (s *ControllerSuite) TestRemoveLastPlayerAbandonsGame() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	err := s.controller.RemovePlayer(s.ctx, game.ID, "player-1")
	s.Require().NoError(err)

	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(model.GameStateAbandoned, updated.State)
}

func (s *ControllerSuite) TestRemovePlayerDuringPlacingAdvancesTurnIfAllPlaced() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1", "player-2"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)
	_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", 'A')
	_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", model.Position{Row: 0, Col: 0})

	// Remove player-2 who hasn't placed yet
	err := s.controller.RemovePlayer(s.ctx, game.ID, "player-2")
	s.Require().NoError(err)

	// Turn should advance since only remaining player has placed
	updated, _ := s.controller.GetGame(s.ctx, game.ID)
	s.Equal(1, updated.CurrentTurn)
	s.Equal(model.GameStateAnnouncing, updated.State)
}

// GetFinalScores tests

func (s *ControllerSuite) TestGetFinalScoresForCompletedGame() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 2)

	// Play through a 2x2 game spelling "CAT" won't fit, but we can test the flow
	positions := []model.Position{
		{Row: 0, Col: 0}, {Row: 0, Col: 1},
		{Row: 1, Col: 0}, {Row: 1, Col: 1},
	}
	letters := []rune{'C', 'A', 'T', 'X'}

	for i, pos := range positions {
		_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", letters[i])
		_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", pos)
	}

	scores, err := s.controller.GetFinalScores(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Len(scores, 1)
	s.Equal(model.PlayerID("player-1"), scores[0].PlayerID)
}

func (s *ControllerSuite) TestGetFinalScoresFailsIfGameNotComplete() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 5)

	_, err := s.controller.GetFinalScores(s.ctx, game.ID)
	s.ErrorIs(err, model.ErrNoGameInProgress)
}

// CreateGameSummary tests

func (s *ControllerSuite) TestCreateGameSummary() {
	s.random.QueueString("GAME12345678")
	players := []model.PlayerID{"player-1"}
	game, _ := s.controller.CreateGame(s.ctx, "LOBBY1", players, 2)

	// Complete the game
	positions := []model.Position{
		{Row: 0, Col: 0}, {Row: 0, Col: 1},
		{Row: 1, Col: 0}, {Row: 1, Col: 1},
	}
	for i, pos := range positions {
		_ = s.controller.AnnounceLetter(s.ctx, game.ID, "player-1", rune('A'+i))
		_ = s.controller.PlaceLetter(s.ctx, game.ID, "player-1", pos)
	}

	summary, err := s.controller.CreateGameSummary(s.ctx, game.ID)
	s.Require().NoError(err)
	s.Equal(game.ID, summary.ID)
	s.Contains(summary.FinalScores, model.PlayerID("player-1"))
}
