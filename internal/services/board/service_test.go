package board

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/mcoot/crosswordgame-go2/internal/testutil"
)

type ServiceSuite struct {
	suite.Suite
	storage *memory.Storage
	service *Service
	ctx     context.Context
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.storage = memory.New()
	s.service = New(s.storage, testutil.NopLogger())
	s.ctx = context.Background()
}

// CreateBoard tests

func (s *ServiceSuite) TestCreateBoardSucceeds() {
	board, err := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	s.Require().NoError(err)

	s.Equal(model.GameID("game-1"), board.GameID)
	s.Equal(model.PlayerID("player-1"), board.PlayerID)
	s.Equal(5, board.Size)
	s.Equal(25, board.EmptyCount())
}

func (s *ServiceSuite) TestCreateBoardIsPersisted() {
	_, err := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	s.Require().NoError(err)

	retrieved, err := s.service.GetBoard(s.ctx, "game-1", "player-1")
	s.Require().NoError(err)
	s.Equal(5, retrieved.Size)
}

// GetBoard tests

func (s *ServiceSuite) TestGetBoardNotFound() {
	_, err := s.service.GetBoard(s.ctx, "game-1", "player-1")
	s.ErrorIs(err, model.ErrBoardNotFound)
}

// GetBoardsForGame tests

func (s *ServiceSuite) TestGetBoardsForGame() {
	_, _ = s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	_, _ = s.service.CreateBoard(s.ctx, "game-1", "player-2", 5)
	_, _ = s.service.CreateBoard(s.ctx, "game-2", "player-1", 5) // Different game

	boards, err := s.service.GetBoardsForGame(s.ctx, "game-1")
	s.Require().NoError(err)
	s.Len(boards, 2)
}

// PlaceLetter tests

func (s *ServiceSuite) TestPlaceLetterSucceeds() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	err := s.service.PlaceLetter(s.ctx, board, 'A', model.Position{Row: 0, Col: 0})
	s.Require().NoError(err)

	s.Equal('A', board.Get(model.Position{Row: 0, Col: 0}))
}

func (s *ServiceSuite) TestPlaceLetterIsPersisted() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	_ = s.service.PlaceLetter(s.ctx, board, 'A', model.Position{Row: 0, Col: 0})

	retrieved, err := s.service.GetBoard(s.ctx, "game-1", "player-1")
	s.Require().NoError(err)
	s.Equal('A', retrieved.Get(model.Position{Row: 0, Col: 0}))
}

func (s *ServiceSuite) TestPlaceLetterNormalizesToUppercase() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	err := s.service.PlaceLetter(s.ctx, board, 'a', model.Position{Row: 0, Col: 0})
	s.Require().NoError(err)

	s.Equal('A', board.Get(model.Position{Row: 0, Col: 0}))
}

func (s *ServiceSuite) TestPlaceLetterInvalidPosition() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	// Out of bounds
	err := s.service.PlaceLetter(s.ctx, board, 'A', model.Position{Row: 5, Col: 0})
	s.ErrorIs(err, model.ErrInvalidPosition)

	err = s.service.PlaceLetter(s.ctx, board, 'A', model.Position{Row: -1, Col: 0})
	s.ErrorIs(err, model.ErrInvalidPosition)
}

func (s *ServiceSuite) TestPlaceLetterCellOccupied() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	_ = s.service.PlaceLetter(s.ctx, board, 'A', model.Position{Row: 0, Col: 0})

	err := s.service.PlaceLetter(s.ctx, board, 'B', model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrCellOccupied)
}

func (s *ServiceSuite) TestPlaceLetterInvalidLetter() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	err := s.service.PlaceLetter(s.ctx, board, '1', model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrInvalidLetter)

	err = s.service.PlaceLetter(s.ctx, board, ' ', model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrInvalidLetter)
}

// ValidatePlacement tests

func (s *ServiceSuite) TestValidatePlacementValid() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	err := s.service.ValidatePlacement(board, model.Position{Row: 0, Col: 0})
	s.NoError(err)

	err = s.service.ValidatePlacement(board, model.Position{Row: 4, Col: 4})
	s.NoError(err)
}

func (s *ServiceSuite) TestValidatePlacementOutOfBounds() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)

	err := s.service.ValidatePlacement(board, model.Position{Row: 5, Col: 0})
	s.ErrorIs(err, model.ErrInvalidPosition)
}

func (s *ServiceSuite) TestValidatePlacementOccupied() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 5)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')

	err := s.service.ValidatePlacement(board, model.Position{Row: 0, Col: 0})
	s.ErrorIs(err, model.ErrCellOccupied)
}

// ValidateLetter tests

func (s *ServiceSuite) TestValidateLetterValid() {
	for letter := 'A'; letter <= 'Z'; letter++ {
		s.NoError(ValidateLetter(letter))
	}
	for letter := 'a'; letter <= 'z'; letter++ {
		s.NoError(ValidateLetter(letter))
	}
}

func (s *ServiceSuite) TestValidateLetterInvalid() {
	s.ErrorIs(ValidateLetter('0'), model.ErrInvalidLetter)
	s.ErrorIs(ValidateLetter(' '), model.ErrInvalidLetter)
	s.ErrorIs(ValidateLetter('@'), model.ErrInvalidLetter)
}

// IsFull tests

func (s *ServiceSuite) TestIsFullEmpty() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 2)
	s.False(s.service.IsFull(board))
}

func (s *ServiceSuite) TestIsFullPartial() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 2)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')
	board.Set(model.Position{Row: 0, Col: 1}, 'B')
	s.False(s.service.IsFull(board))
}

func (s *ServiceSuite) TestIsFullComplete() {
	board, _ := s.service.CreateBoard(s.ctx, "game-1", "player-1", 2)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')
	board.Set(model.Position{Row: 0, Col: 1}, 'B')
	board.Set(model.Position{Row: 1, Col: 0}, 'C')
	board.Set(model.Position{Row: 1, Col: 1}, 'D')
	s.True(s.service.IsFull(board))
}
