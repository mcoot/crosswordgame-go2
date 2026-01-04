package scoring

import (
	"testing"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite
	dictService *dictionary.Service
	service     *Service
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	storage := memory.New()
	s.dictService = dictionary.New(storage)
	s.service = New(s.dictService)
}

func (s *ServiceSuite) loadDictionary(words []string) {
	err := s.dictService.LoadWords(words)
	s.Require().NoError(err)
}

// Helper to create a board with letters
func (s *ServiceSuite) createBoard(size int, rows ...string) *model.Board {
	board := model.NewBoard("game-1", "player-1", size)
	for row, letters := range rows {
		for col, letter := range letters {
			if letter != ' ' && letter != '.' {
				board.Set(model.Position{Row: row, Col: col}, letter)
			}
		}
	}
	return board
}

// Basic scoring tests

func (s *ServiceSuite) TestScoreEmptyBoard() {
	s.loadDictionary([]string{"test"})
	board := s.createBoard(3, "...", "...", "...")

	result := s.service.ScoreBoard(board)

	s.Equal(model.PlayerID("player-1"), result.PlayerID)
	s.Empty(result.Words)
	s.Equal(0, result.TotalScore)
}

func (s *ServiceSuite) TestScoreSingleHorizontalWord() {
	s.loadDictionary([]string{"cat"})
	board := s.createBoard(3,
		"CAT",
		"...",
		"...",
	)

	result := s.service.ScoreBoard(board)

	s.Len(result.Words, 1)
	s.Equal("CAT", result.Words[0].Word)
	s.True(result.Words[0].Horizontal)
	s.Equal(model.Position{Row: 0, Col: 0}, result.Words[0].StartPos)
	s.Equal(3, result.Words[0].Length)
	s.Equal(6, result.Words[0].Score) // Full row bonus: 3 * 2
	s.Equal(6, result.TotalScore)
}

func (s *ServiceSuite) TestScoreSingleVerticalWord() {
	s.loadDictionary([]string{"cat"})
	board := s.createBoard(3,
		"C..",
		"A..",
		"T..",
	)

	result := s.service.ScoreBoard(board)

	s.Len(result.Words, 1)
	s.Equal("CAT", result.Words[0].Word)
	s.False(result.Words[0].Horizontal)
	s.Equal(model.Position{Row: 0, Col: 0}, result.Words[0].StartPos)
	s.Equal(3, result.Words[0].Length)
	s.Equal(6, result.Words[0].Score) // Full column bonus
}

func (s *ServiceSuite) TestScorePartialLineNoBonus() {
	s.loadDictionary([]string{"cat"})
	board := s.createBoard(5,
		"CAT..",
		".....",
		".....",
		".....",
		".....",
	)

	result := s.service.ScoreBoard(board)

	s.Len(result.Words, 1)
	s.Equal(3, result.Words[0].Score) // No bonus: not full row
	s.Equal(3, result.TotalScore)
}

func (s *ServiceSuite) TestScoreMultipleWordsInRow() {
	s.loadDictionary([]string{"at", "be"})
	board := s.createBoard(5,
		"ATXBE",
		".....",
		".....",
		".....",
		".....",
	)

	result := s.service.ScoreBoard(board)

	// Should find both "AT" and "BE"
	s.Len(result.Words, 2)
	s.Equal(4, result.TotalScore) // 2 + 2
}

func (s *ServiceSuite) TestScoreOverlappingWordsPicksLonger() {
	// "CAT" and "AT" overlap - should pick CAT (longer)
	s.loadDictionary([]string{"cat", "at"})
	board := s.createBoard(3,
		"CAT",
		"...",
		"...",
	)

	result := s.service.ScoreBoard(board)

	// Should only have CAT, not AT (they overlap)
	s.Len(result.Words, 1)
	s.Equal("CAT", result.Words[0].Word)
	s.Equal(6, result.TotalScore)
}

func (s *ServiceSuite) TestScoreSameLetterInBothDirections() {
	// Same letter can score in both horizontal and vertical
	s.loadDictionary([]string{"cat", "cup"})
	board := s.createBoard(3,
		"CAT",
		"U..",
		"P..",
	)

	result := s.service.ScoreBoard(board)

	// Should find CAT (horizontal) and CUP (vertical)
	s.Len(result.Words, 2)
	s.Equal(12, result.TotalScore) // 6 + 6 (both full lines)
}

func (s *ServiceSuite) TestScoreComplexBoard() {
	s.loadDictionary([]string{"hello", "world", "hi", "we"})
	board := s.createBoard(5,
		"HELLO",
		"I....",
		".W...",
		".E...",
		"WORLD",
	)

	result := s.service.ScoreBoard(board)

	// Find all expected words
	foundHello := false
	foundWorld := false
	foundHi := false
	foundWe := false

	for _, w := range result.Words {
		switch w.Word {
		case "HELLO":
			foundHello = true
			s.True(w.Horizontal)
			s.Equal(10, w.Score) // Full row bonus
		case "WORLD":
			foundWorld = true
			s.True(w.Horizontal)
			s.Equal(10, w.Score) // Full row bonus
		case "HI":
			foundHi = true
			s.False(w.Horizontal)
			s.Equal(2, w.Score)
		case "WE":
			foundWe = true
			s.False(w.Horizontal)
			s.Equal(2, w.Score)
		}
	}

	s.True(foundHello, "should find HELLO")
	s.True(foundWorld, "should find WORLD")
	s.True(foundHi, "should find HI")
	s.True(foundWe, "should find WE")
	s.Equal(24, result.TotalScore)
}

// ScoreMultipleBoards tests

func (s *ServiceSuite) TestScoreMultipleBoardsSorted() {
	s.loadDictionary([]string{"cat", "dog", "hi"})

	board1 := model.NewBoard("game-1", "player-1", 3)
	board1.Set(model.Position{Row: 0, Col: 0}, 'C')
	board1.Set(model.Position{Row: 0, Col: 1}, 'A')
	board1.Set(model.Position{Row: 0, Col: 2}, 'T')
	// Score: 6 (CAT full row)

	board2 := model.NewBoard("game-1", "player-2", 3)
	board2.Set(model.Position{Row: 0, Col: 0}, 'D')
	board2.Set(model.Position{Row: 0, Col: 1}, 'O')
	board2.Set(model.Position{Row: 0, Col: 2}, 'G')
	board2.Set(model.Position{Row: 1, Col: 0}, 'H')
	board2.Set(model.Position{Row: 2, Col: 0}, 'I')
	// Score: 6 (DOG) + 2 (HI, not full column in a 3x3) = 8
	// Actually HI is in column 0, rows 1-2, so it's not a full column

	boards := []*model.Board{board1, board2}
	scores := s.service.ScoreMultipleBoards(boards)

	s.Len(scores, 2)
	// Sorted by score descending
	s.Equal(model.PlayerID("player-2"), scores[0].PlayerID)
	s.Equal(model.PlayerID("player-1"), scores[1].PlayerID)
}

// DetermineWinner tests

func (s *ServiceSuite) TestDetermineWinnerClearWinner() {
	scores := []model.BoardScore{
		{PlayerID: "player-1", TotalScore: 20},
		{PlayerID: "player-2", TotalScore: 15},
	}

	winner := s.service.DetermineWinner(scores)
	s.Equal(model.PlayerID("player-1"), winner)
}

func (s *ServiceSuite) TestDetermineWinnerTie() {
	scores := []model.BoardScore{
		{PlayerID: "player-1", TotalScore: 20},
		{PlayerID: "player-2", TotalScore: 20},
	}

	winner := s.service.DetermineWinner(scores)
	s.Empty(winner)
}

func (s *ServiceSuite) TestDetermineWinnerEmpty() {
	scores := []model.BoardScore{}
	winner := s.service.DetermineWinner(scores)
	s.Empty(winner)
}

// Edge cases

func (s *ServiceSuite) TestScoreDictionaryNotLoaded() {
	// Don't load dictionary
	board := s.createBoard(3, "CAT", "...", "...")

	result := s.service.ScoreBoard(board)

	s.Empty(result.Words)
	s.Equal(0, result.TotalScore)
}

func (s *ServiceSuite) TestScoreTwoLetterWord() {
	s.loadDictionary([]string{"at"})
	board := s.createBoard(3,
		"AT.",
		"...",
		"...",
	)

	result := s.service.ScoreBoard(board)

	s.Len(result.Words, 1)
	s.Equal("AT", result.Words[0].Word)
	s.Equal(2, result.Words[0].Score)
}

func (s *ServiceSuite) TestScoreNoValidWords() {
	s.loadDictionary([]string{"hello", "world"})
	board := s.createBoard(3,
		"XYZ",
		"ABC",
		"QRS",
	)

	result := s.service.ScoreBoard(board)

	s.Empty(result.Words)
	s.Equal(0, result.TotalScore)
}
