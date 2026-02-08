package bot_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
	"github.com/mcoot/crosswordgame-go2/internal/testutil"
)

type LLMStrategySuite struct {
	suite.Suite
	mockLLM    *mocks.MockLLMClient
	mockRandom *mocks.MockRandom
	mockClock  *mocks.MockClock
	strategy   *bot.LLMStrategy
	ctx        context.Context
}

func TestLLMStrategySuite(t *testing.T) {
	suite.Run(t, new(LLMStrategySuite))
}

func (s *LLMStrategySuite) SetupTest() {
	s.mockLLM = mocks.NewMockLLMClient()
	s.mockRandom = mocks.NewMockRandom()
	s.mockClock = mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	fallback := bot.NewRandomStrategy(s.mockRandom)
	s.strategy = bot.NewLLMStrategy(s.mockLLM, fallback, s.mockClock, testutil.NopLogger(), 100)
	s.ctx = context.Background()
}

func (s *LLMStrategySuite) game() *model.Game {
	return &model.Game{
		GridSize:      5,
		CurrentTurn:   3,
		CurrentLetter: 'S',
		Players:       []model.PlayerID{"p1"},
	}
}

func (s *LLMStrategySuite) board() *model.Board {
	b := model.NewBoard("game1", "p1", 5)
	b.Set(model.Position{Row: 0, Col: 1}, 'E')
	b.Set(model.Position{Row: 0, Col: 3}, 'T')
	b.Set(model.Position{Row: 1, Col: 0}, 'H')
	return b
}

// --- ChooseLetter tests ---

func (s *LLMStrategySuite) TestChooseLetter_Success() {
	s.mockLLM.QueueResponse(`{"letter":"S"}`)

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('S', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_LowercaseNormalized() {
	s.mockLLM.QueueResponse(`{"letter":"s"}`)

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('S', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_CodeFenceWrapped() {
	s.mockLLM.QueueResponse("```json\n{\"letter\":\"R\"}\n```")

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('R', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_InvalidJSON_FallsBackToRandom() {
	s.mockLLM.QueueResponse("not json")
	s.mockRandom.QueueIntn(4) // 'E'

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('E', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_InvalidLetter_FallsBackToRandom() {
	s.mockLLM.QueueResponse(`{"letter":"3"}`)
	s.mockRandom.QueueIntn(0) // 'A'

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('A', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_MultipleChars_FallsBackToRandom() {
	s.mockLLM.QueueResponse(`{"letter":"AB"}`)
	s.mockRandom.QueueIntn(1) // 'B'

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('B', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_LLMError_FallsBackToRandom() {
	s.mockLLM.QueueError(fmt.Errorf("connection refused"))
	s.mockRandom.QueueIntn(2) // 'C'

	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('C', letter)
}

func (s *LLMStrategySuite) TestChooseLetter_DailyLimitReached_FallsBackToRandom() {
	// Exhaust the daily limit
	for range 100 {
		s.mockLLM.QueueResponse(`{"letter":"A"}`)
	}
	for range 100 {
		s.strategy.ChooseLetter(s.ctx, s.game())
	}

	// 101st call should fall back
	s.mockRandom.QueueIntn(25) // 'Z'
	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('Z', letter)
}

// --- ChoosePosition tests ---

func (s *LLMStrategySuite) TestChoosePosition_Success() {
	s.mockLLM.QueueResponse(`{"row":2,"col":3}`)

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	s.Equal(model.Position{Row: 2, Col: 3}, pos)
}

func (s *LLMStrategySuite) TestChoosePosition_CodeFenceWrapped() {
	s.mockLLM.QueueResponse("```json\n{\"row\":4,\"col\":4}\n```")

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	s.Equal(model.Position{Row: 4, Col: 4}, pos)
}

func (s *LLMStrategySuite) TestChoosePosition_InvalidJSON_FallsBackToRandom() {
	s.mockLLM.QueueResponse("row 2, col 3")
	s.mockRandom.QueueIntn(0) // picks first empty

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	// First empty cell in 5x5 with E at (0,1), T at (0,3), H at (1,0) is (0,0)
	s.Equal(model.Position{Row: 0, Col: 0}, pos)
}

func (s *LLMStrategySuite) TestChoosePosition_OutOfBounds_FallsBackToRandom() {
	s.mockLLM.QueueResponse(`{"row":10,"col":0}`)
	s.mockRandom.QueueIntn(0)

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	s.Equal(model.Position{Row: 0, Col: 0}, pos)
}

func (s *LLMStrategySuite) TestChoosePosition_OccupiedCell_FallsBackToRandom() {
	s.mockLLM.QueueResponse(`{"row":0,"col":1}`) // 'E' is already there
	s.mockRandom.QueueIntn(0)

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	s.Equal(model.Position{Row: 0, Col: 0}, pos)
}

func (s *LLMStrategySuite) TestChoosePosition_LLMError_FallsBackToRandom() {
	s.mockLLM.QueueError(fmt.Errorf("timeout"))
	s.mockRandom.QueueIntn(0)

	pos := s.strategy.ChoosePosition(s.ctx, s.game(), s.board())
	s.Equal(model.Position{Row: 0, Col: 0}, pos)
}

// --- Daily limit reset test ---

func (s *LLMStrategySuite) TestDailyLimit_ResetsOnNewDay() {
	// Exhaust the limit
	for range 100 {
		s.mockLLM.QueueResponse(`{"letter":"A"}`)
	}
	for range 100 {
		s.strategy.ChooseLetter(s.ctx, s.game())
	}

	// Verify limit is hit
	s.mockRandom.QueueIntn(0)
	letter := s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('A', letter) // from fallback random (Intn 0 -> 'A')

	// Advance clock to next day
	s.mockClock.Advance(24 * time.Hour)

	// Should work again
	s.mockLLM.QueueResponse(`{"letter":"Z"}`)
	letter = s.strategy.ChooseLetter(s.ctx, s.game())
	s.Equal('Z', letter)
}

// --- Prompt content test ---

func (s *LLMStrategySuite) TestPositionPrompt_ContainsBoardState() {
	s.mockLLM.QueueResponse(`{"row":0,"col":0}`)

	s.strategy.ChoosePosition(s.ctx, s.game(), s.board())

	s.Require().Len(s.mockLLM.Prompts, 1)
	prompt := s.mockLLM.Prompts[0]
	s.Contains(prompt, "E")
	s.Contains(prompt, "T")
	s.Contains(prompt, "H")
	s.Contains(prompt, "'S'")
	s.Contains(prompt, "empty")
}

func (s *LLMStrategySuite) TestLetterPrompt_ContainsGameContext() {
	s.mockLLM.QueueResponse(`{"letter":"A"}`)

	s.strategy.ChooseLetter(s.ctx, s.game())

	s.Require().Len(s.mockLLM.Prompts, 1)
	prompt := s.mockLLM.Prompts[0]
	s.Contains(prompt, "5x5")
	s.Contains(prompt, "4 of 25")
}
