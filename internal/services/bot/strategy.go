package bot

import "github.com/mcoot/crosswordgame-go2/internal/model"

// Strategy defines how a bot chooses letters and positions
type Strategy interface {
	// ChooseLetter selects a letter to announce
	ChooseLetter(game *model.Game) rune
	// ChoosePosition selects a position to place a letter on the board
	ChoosePosition(game *model.Game, board *model.Board) model.Position
}
