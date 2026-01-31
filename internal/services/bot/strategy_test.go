package bot_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
)

type StrategySuite struct {
	suite.Suite
	mockRandom *mocks.MockRandom
	strategy   *bot.RandomStrategy
}

func TestStrategySuite(t *testing.T) {
	suite.Run(t, new(StrategySuite))
}

func (s *StrategySuite) SetupTest() {
	s.mockRandom = mocks.NewMockRandom()
	s.strategy = bot.NewRandomStrategy(s.mockRandom)
}

func (s *StrategySuite) TestChooseLetter_ReturnsValidLetter() {
	s.mockRandom.QueueIntn(0) // 'A'
	letter := s.strategy.ChooseLetter(&model.Game{})
	s.Equal('A', letter)

	s.mockRandom.QueueIntn(25) // 'Z'
	letter = s.strategy.ChooseLetter(&model.Game{})
	s.Equal('Z', letter)

	s.mockRandom.QueueIntn(12) // 'M'
	letter = s.strategy.ChooseLetter(&model.Game{})
	s.Equal('M', letter)
}

func (s *StrategySuite) TestChoosePosition_EmptyBoard() {
	board := model.NewBoard("game1", "player1", 3)
	// 9 empty cells, random picks index 4
	s.mockRandom.QueueIntn(4)

	pos := s.strategy.ChoosePosition(&model.Game{}, board)
	// Index 4 = (1, 1) in a 3x3 grid
	s.Equal(model.Position{Row: 1, Col: 1}, pos)
}

func (s *StrategySuite) TestChoosePosition_PartiallyFilledBoard() {
	board := model.NewBoard("game1", "player1", 2)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')
	board.Set(model.Position{Row: 0, Col: 1}, 'B')
	// Only (1,0) and (1,1) are empty
	s.mockRandom.QueueIntn(1) // Pick second empty cell

	pos := s.strategy.ChoosePosition(&model.Game{}, board)
	s.Equal(model.Position{Row: 1, Col: 1}, pos)
}

func (s *StrategySuite) TestChoosePosition_OnlyOneEmpty() {
	board := model.NewBoard("game1", "player1", 2)
	board.Set(model.Position{Row: 0, Col: 0}, 'A')
	board.Set(model.Position{Row: 0, Col: 1}, 'B')
	board.Set(model.Position{Row: 1, Col: 0}, 'C')
	// Only (1,1) is empty
	s.mockRandom.QueueIntn(0)

	pos := s.strategy.ChoosePosition(&model.Game{}, board)
	s.Equal(model.Position{Row: 1, Col: 1}, pos)
}
