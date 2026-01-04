package model

// Position identifies a cell on the board
type Position struct {
	Row int // 0-indexed from top
	Col int // 0-indexed from left
}

// Board represents a player's grid for a specific game
type Board struct {
	GameID   GameID
	PlayerID PlayerID
	Size     int      // Grid dimension (e.g., 5 for 5x5)
	Cells    [][]rune // Row-major: Cells[row][col], 0 means empty
}

// NewBoard creates an empty board of the given size
func NewBoard(gameID GameID, playerID PlayerID, size int) *Board {
	cells := make([][]rune, size)
	for i := range cells {
		cells[i] = make([]rune, size)
	}
	return &Board{
		GameID:   gameID,
		PlayerID: playerID,
		Size:     size,
		Cells:    cells,
	}
}

// Get returns the letter at the given position, or 0 if empty
func (b *Board) Get(pos Position) rune {
	if !b.IsValidPosition(pos) {
		return 0
	}
	return b.Cells[pos.Row][pos.Col]
}

// Set places a letter at the given position
func (b *Board) Set(pos Position, letter rune) {
	if b.IsValidPosition(pos) {
		b.Cells[pos.Row][pos.Col] = letter
	}
}

// IsEmpty returns true if the cell at the given position is empty
func (b *Board) IsEmpty(pos Position) bool {
	return b.Get(pos) == 0
}

// IsValidPosition returns true if the position is within bounds
func (b *Board) IsValidPosition(pos Position) bool {
	return pos.Row >= 0 && pos.Row < b.Size && pos.Col >= 0 && pos.Col < b.Size
}

// IsFull returns true if all cells are filled
func (b *Board) IsFull() bool {
	for row := 0; row < b.Size; row++ {
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] == 0 {
				return false
			}
		}
	}
	return true
}

// EmptyCount returns the number of empty cells
func (b *Board) EmptyCount() int {
	count := 0
	for row := 0; row < b.Size; row++ {
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] == 0 {
				count++
			}
		}
	}
	return count
}

// GetRow returns all letters in the given row
func (b *Board) GetRow(row int) []rune {
	if row < 0 || row >= b.Size {
		return nil
	}
	result := make([]rune, b.Size)
	copy(result, b.Cells[row])
	return result
}

// GetCol returns all letters in the given column
func (b *Board) GetCol(col int) []rune {
	if col < 0 || col >= b.Size {
		return nil
	}
	result := make([]rune, b.Size)
	for row := 0; row < b.Size; row++ {
		result[row] = b.Cells[row][col]
	}
	return result
}

// WordMatch represents a valid word found on the board
type WordMatch struct {
	Word       string
	StartPos   Position
	Horizontal bool // true = left-to-right, false = top-to-bottom
	Length     int
	Score      int // Calculated score for this word
}

// BoardScore is the complete scoring result for a board
type BoardScore struct {
	PlayerID   PlayerID
	Words      []WordMatch
	TotalScore int
}
