package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/llm"
	"github.com/mcoot/crosswordgame-go2/internal/model"
)

const (
	llmCallTimeout       = 3 * time.Second
	defaultLLMDailyLimit = 5000
)

// LLMStrategy uses an LLM to make word-aware letter and position choices,
// falling back to random on any failure or rate limit
type LLMStrategy struct {
	client   llm.Client
	fallback *RandomStrategy
	clock    clock.Clock
	logger   *slog.Logger

	dailyLimit   int
	mu           sync.Mutex
	callCount    int
	callCountDay string
}

// NewLLMStrategy creates a new LLMStrategy
func NewLLMStrategy(client llm.Client, fallback *RandomStrategy, clk clock.Clock, logger *slog.Logger, dailyLimit int) *LLMStrategy {
	if dailyLimit <= 0 {
		dailyLimit = defaultLLMDailyLimit
	}
	return &LLMStrategy{
		client:     client,
		fallback:   fallback,
		clock:      clk,
		logger:     logger.With(slog.String("component", "llm-strategy")),
		dailyLimit: dailyLimit,
	}
}

// ChooseLetter asks the LLM to pick a useful letter, falling back to random on failure
func (s *LLMStrategy) ChooseLetter(ctx context.Context, game *model.Game) rune {
	if !s.tryIncrementCallCount() {
		s.logger.Warn("LLM daily limit reached, falling back to random")
		return s.fallback.ChooseLetter(ctx, game)
	}

	prompt := s.buildLetterPrompt(game)

	llmCtx, cancel := context.WithTimeout(ctx, llmCallTimeout)
	defer cancel()

	resp, err := s.client.GenerateContent(llmCtx, prompt)
	if err != nil {
		s.logger.Warn("LLM call failed for letter selection", slog.String("error", err.Error()))
		return s.fallback.ChooseLetter(ctx, game)
	}

	letter, err := parseLetterResponse(resp.Text)
	if err != nil {
		s.logger.Warn("LLM returned invalid letter", slog.String("error", err.Error()), slog.String("response", resp.Text))
		return s.fallback.ChooseLetter(ctx, game)
	}

	return letter
}

// ChoosePosition asks the LLM to place a letter strategically, falling back to random on failure
func (s *LLMStrategy) ChoosePosition(ctx context.Context, game *model.Game, board *model.Board) model.Position {
	if !s.tryIncrementCallCount() {
		s.logger.Warn("LLM daily limit reached, falling back to random")
		return s.fallback.ChoosePosition(ctx, game, board)
	}

	prompt := s.buildPositionPrompt(game, board)

	llmCtx, cancel := context.WithTimeout(ctx, llmCallTimeout)
	defer cancel()

	resp, err := s.client.GenerateContent(llmCtx, prompt)
	if err != nil {
		s.logger.Warn("LLM call failed for position selection", slog.String("error", err.Error()))
		return s.fallback.ChoosePosition(ctx, game, board)
	}

	pos, err := parsePositionResponse(resp.Text, board)
	if err != nil {
		s.logger.Warn("LLM returned invalid position", slog.String("error", err.Error()), slog.String("response", resp.Text))
		return s.fallback.ChoosePosition(ctx, game, board)
	}

	return pos
}

// tryIncrementCallCount atomically checks and increments the daily call counter.
// Returns false if the daily limit has been reached.
func (s *LLMStrategy) tryIncrementCallCount() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := s.clock.Now().UTC().Format("2006-01-02")
	if today != s.callCountDay {
		s.callCount = 0
		s.callCountDay = today
	}

	if s.callCount >= s.dailyLimit {
		return false
	}
	s.callCount++
	return true
}

func (s *LLMStrategy) buildLetterPrompt(game *model.Game) string {
	// We don't have the bot's board during letter selection (it's chosen before placement),
	// so we provide game context only
	return fmt.Sprintf(
		"Word grid game. %dx%d board, turn %d of %d. "+
			"Pick a letter A-Z that helps form English words in rows/columns.\n"+
			"Reply ONLY: {\"letter\":\"X\"}",
		game.GridSize, game.GridSize, game.CurrentTurn+1, game.TotalTurns(),
	)
}

func (s *LLMStrategy) buildPositionPrompt(game *model.Game, board *model.Board) string {
	var sb strings.Builder
	sb.WriteString("Word grid game. Your board:\n")

	// Column headers
	sb.WriteString("  ")
	for col := 0; col < board.Size; col++ {
		fmt.Fprintf(&sb, " %d", col)
	}
	sb.WriteByte('\n')

	// Board rows
	for row := 0; row < board.Size; row++ {
		fmt.Fprintf(&sb, "%d:", row)
		for col := 0; col < board.Size; col++ {
			cell := board.Cells[row][col]
			if cell == 0 {
				sb.WriteString(" .")
			} else {
				fmt.Fprintf(&sb, " %c", cell)
			}
		}
		sb.WriteByte('\n')
	}

	fmt.Fprintf(&sb,
		"Place letter '%c' in an empty cell (.) to help form words.\n"+
			"Reply ONLY: {\"row\":R,\"col\":C}",
		game.CurrentLetter,
	)

	return sb.String()
}

type letterResponse struct {
	Letter string `json:"letter"`
}

type positionResponse struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// stripCodeFences removes markdown code fences that LLMs commonly wrap around JSON
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (with optional language tag)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func parseLetterResponse(raw string) (rune, error) {
	cleaned := stripCodeFences(raw)
	var resp letterResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return 0, fmt.Errorf("parsing letter JSON: %w", err)
	}

	if len(resp.Letter) != 1 {
		return 0, fmt.Errorf("expected single letter, got %q", resp.Letter)
	}

	letter := unicode.ToUpper(rune(resp.Letter[0]))
	if letter < 'A' || letter > 'Z' {
		return 0, fmt.Errorf("letter %q not in A-Z range", resp.Letter)
	}

	return letter, nil
}

func parsePositionResponse(raw string, board *model.Board) (model.Position, error) {
	cleaned := stripCodeFences(raw)
	var resp positionResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return model.Position{}, fmt.Errorf("parsing position JSON: %w", err)
	}

	pos := model.Position{Row: resp.Row, Col: resp.Col}

	if !board.IsValidPosition(pos) {
		return model.Position{}, fmt.Errorf("position (%d,%d) out of bounds for %dx%d board", resp.Row, resp.Col, board.Size, board.Size)
	}

	if !board.IsEmpty(pos) {
		return model.Position{}, fmt.Errorf("position (%d,%d) is already occupied", resp.Row, resp.Col)
	}

	return pos, nil
}
