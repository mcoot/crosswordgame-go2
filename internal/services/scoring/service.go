package scoring

import (
	"sort"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
)

// Service provides scoring functionality for completed boards
type Service struct {
	dictionary *dictionary.Service
}

// New creates a new ScoringService
func New(dictionary *dictionary.Service) *Service {
	return &Service{
		dictionary: dictionary,
	}
}

// ScoreBoard calculates the final score for a completed board
func (s *Service) ScoreBoard(board *model.Board) *model.BoardScore {
	result := &model.BoardScore{
		PlayerID: board.PlayerID,
		Words:    []model.WordMatch{},
	}

	// Find words in rows (horizontal)
	for row := 0; row < board.Size; row++ {
		letters := board.GetRow(row)
		words := s.findBestWordsInLine(letters, board.Size)
		for _, w := range words {
			result.Words = append(result.Words, model.WordMatch{
				Word:       w.word,
				StartPos:   model.Position{Row: row, Col: w.start},
				Horizontal: true,
				Length:     w.length,
				Score:      w.score,
			})
			result.TotalScore += w.score
		}
	}

	// Find words in columns (vertical)
	for col := 0; col < board.Size; col++ {
		letters := board.GetCol(col)
		words := s.findBestWordsInLine(letters, board.Size)
		for _, w := range words {
			result.Words = append(result.Words, model.WordMatch{
				Word:       w.word,
				StartPos:   model.Position{Row: w.start, Col: col},
				Horizontal: false,
				Length:     w.length,
				Score:      w.score,
			})
			result.TotalScore += w.score
		}
	}

	return result
}

// wordCandidate represents a potential word found in a line
type wordCandidate struct {
	word   string
	start  int
	length int
	score  int
}

// findBestWordsInLine finds the best non-overlapping set of words in a line
// Uses greedy algorithm: prefer longer words first
func (s *Service) findBestWordsInLine(letters []rune, gridSize int) []wordCandidate {
	// Find all valid words
	validWords := s.dictionary.FindAllValidWords(letters)
	if len(validWords) == 0 {
		return nil
	}

	// Convert to candidates with scores
	candidates := make([]wordCandidate, 0, len(validWords))
	for _, vw := range validWords {
		length := vw.End - vw.Start
		score := length
		if length == gridSize {
			score = length * 2 // Full line bonus
		}
		candidates = append(candidates, wordCandidate{
			word:   vw.Word,
			start:  vw.Start,
			length: length,
			score:  score,
		})
	}

	// Sort by length descending (greedy: prefer longer words)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].length > candidates[j].length
	})

	// Greedy selection: pick non-overlapping words
	used := make([]bool, len(letters))
	var selected []wordCandidate

	for _, c := range candidates {
		// Check if any position in this word is already used
		overlaps := false
		for i := c.start; i < c.start+c.length; i++ {
			if used[i] {
				overlaps = true
				break
			}
		}

		if !overlaps {
			selected = append(selected, c)
			// Mark positions as used
			for i := c.start; i < c.start+c.length; i++ {
				used[i] = true
			}
		}
	}

	return selected
}

// ScoreMultipleBoards scores all boards and returns results sorted by score
func (s *Service) ScoreMultipleBoards(boards []*model.Board) []model.BoardScore {
	scores := make([]model.BoardScore, 0, len(boards))
	for _, board := range boards {
		scores = append(scores, *s.ScoreBoard(board))
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	return scores
}

// DetermineWinner returns the winner's PlayerID, or empty string if tie
func (s *Service) DetermineWinner(scores []model.BoardScore) model.PlayerID {
	if len(scores) == 0 {
		return ""
	}

	topScore := scores[0].TotalScore
	tieCount := 0
	for _, score := range scores {
		if score.TotalScore == topScore {
			tieCount++
		}
	}

	if tieCount > 1 {
		return "" // Tie
	}

	return scores[0].PlayerID
}

// Interface for dependency injection
type ServiceInterface interface {
	ScoreBoard(board *model.Board) *model.BoardScore
	ScoreMultipleBoards(boards []*model.Board) []model.BoardScore
	DetermineWinner(scores []model.BoardScore) model.PlayerID
}

var _ ServiceInterface = (*Service)(nil)
