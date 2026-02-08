package bot

import (
	"context"

	"github.com/mcoot/crosswordgame-go2/internal/model"
)

// Strategy defines how a bot chooses letters and positions
type Strategy interface {
	// ChooseLetter selects a letter to announce
	ChooseLetter(ctx context.Context, game *model.Game) rune
	// ChoosePosition selects a position to place a letter on the board
	ChoosePosition(ctx context.Context, game *model.Game, board *model.Board) model.Position
}
