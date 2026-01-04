package board

import (
	"context"
	"unicode"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Service provides board operations
type Service struct {
	storage storage.Storage
}

// New creates a new BoardService
func New(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
	}
}

// CreateBoard initializes an empty board for a player in a game
func (s *Service) CreateBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID, size int) (*model.Board, error) {
	board := model.NewBoard(gameID, playerID, size)
	if err := s.storage.SaveBoard(ctx, board); err != nil {
		return nil, err
	}
	return board, nil
}

// GetBoard retrieves a player's board
func (s *Service) GetBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID) (*model.Board, error) {
	return s.storage.GetBoard(ctx, gameID, playerID)
}

// GetBoardsForGame retrieves all boards for a game
func (s *Service) GetBoardsForGame(ctx context.Context, gameID model.GameID) ([]*model.Board, error) {
	return s.storage.GetBoardsForGame(ctx, gameID)
}

// PlaceLetter places a letter at the specified position on a board
func (s *Service) PlaceLetter(ctx context.Context, board *model.Board, letter rune, pos model.Position) error {
	if err := s.ValidatePlacement(board, pos); err != nil {
		return err
	}
	if err := ValidateLetter(letter); err != nil {
		return err
	}

	board.Set(pos, unicode.ToUpper(letter))
	return s.storage.SaveBoard(ctx, board)
}

// ValidatePlacement checks if a position is valid and empty
func (s *Service) ValidatePlacement(board *model.Board, pos model.Position) error {
	if !board.IsValidPosition(pos) {
		return model.ErrInvalidPosition
	}
	if !board.IsEmpty(pos) {
		return model.ErrCellOccupied
	}
	return nil
}

// ValidateLetter checks if a letter is a valid A-Z character
func ValidateLetter(letter rune) error {
	upper := unicode.ToUpper(letter)
	if upper < 'A' || upper > 'Z' {
		return model.ErrInvalidLetter
	}
	return nil
}

// IsFull checks if all cells are filled
func (s *Service) IsFull(board *model.Board) bool {
	return board.IsFull()
}

// Interface for dependency injection
type ServiceInterface interface {
	CreateBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID, size int) (*model.Board, error)
	GetBoard(ctx context.Context, gameID model.GameID, playerID model.PlayerID) (*model.Board, error)
	GetBoardsForGame(ctx context.Context, gameID model.GameID) ([]*model.Board, error)
	PlaceLetter(ctx context.Context, board *model.Board, letter rune, pos model.Position) error
	ValidatePlacement(board *model.Board, pos model.Position) error
	IsFull(board *model.Board) bool
}

var _ ServiceInterface = (*Service)(nil)
