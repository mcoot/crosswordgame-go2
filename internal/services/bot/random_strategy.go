package bot

import (
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// RandomStrategy picks random letters and random empty positions
type RandomStrategy struct {
	random random.Random
}

// NewRandomStrategy creates a new RandomStrategy
func NewRandomStrategy(rnd random.Random) *RandomStrategy {
	return &RandomStrategy{random: rnd}
}

// ChooseLetter returns a random uppercase letter A-Z
func (s *RandomStrategy) ChooseLetter(game *model.Game) rune {
	return rune('A' + s.random.Intn(26))
}

// ChoosePosition picks a random empty cell on the board
func (s *RandomStrategy) ChoosePosition(game *model.Game, board *model.Board) model.Position {
	var empty []model.Position
	for row := 0; row < board.Size; row++ {
		for col := 0; col < board.Size; col++ {
			if board.Cells[row][col] == 0 {
				empty = append(empty, model.Position{Row: row, Col: col})
			}
		}
	}
	if len(empty) == 0 {
		return model.Position{Row: 0, Col: 0}
	}
	return empty[s.random.Intn(len(empty))]
}
