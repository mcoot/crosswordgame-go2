package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Output handles formatting output based on the configured format
type Output struct {
	format string
}

// NewOutput creates a new Output formatter
func NewOutput(format string) *Output {
	return &Output{format: format}
}

// Print outputs data in the configured format
func (o *Output) Print(data any) {
	if o.format == "json" {
		o.printJSON(data)
	} else {
		o.printText(data)
	}
}

// PrintError outputs an error
func (o *Output) PrintError(err error) {
	if o.format == "json" {
		errData := map[string]any{
			"error": map[string]string{
				"message": err.Error(),
			},
		}
		data, _ := json.Marshal(errData)
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}

// PrintMessage outputs a simple message
func (o *Output) PrintMessage(msg string) {
	if o.format == "json" {
		data, _ := json.Marshal(map[string]string{"message": msg})
		fmt.Println(string(data))
	} else {
		fmt.Println(msg)
	}
}

func (o *Output) printJSON(data any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func (o *Output) printText(data any) {
	switch v := data.(type) {
	case Player:
		o.printPlayer(v)
	case AuthResult:
		o.printAuthResult(v)
	case Lobby:
		o.printLobby(v)
	case LobbyConfig:
		o.printLobbyConfig(v)
	case GameState:
		o.printGameState(v)
	case AnnounceResult:
		o.printAnnounceResult(v)
	case PlaceResult:
		o.printPlaceResult(v)
	case HealthResult:
		o.printHealthResult(v)
	default:
		// Fallback to JSON for unknown types
		o.printJSON(data)
	}
}

// Player response type (matches API)
type Player struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	IsGuest     bool   `json:"is_guest"`
}

// AuthResult combines player and token
type AuthResult struct {
	Player       Player `json:"player"`
	SessionToken string `json:"session_token"`
}

// Lobby response type
type Lobby struct {
	Code        string        `json:"code"`
	State       string        `json:"state"`
	Config      LobbyConfig   `json:"config"`
	Members     []LobbyMember `json:"members"`
	CurrentGame *string       `json:"current_game"`
}

// LobbyConfig response type
type LobbyConfig struct {
	GridSize int `json:"grid_size"`
}

// LobbyMember response type
type LobbyMember struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsHost      bool   `json:"is_host"`
}

// GameState response type
type GameState struct {
	ID               string            `json:"id"`
	State            string            `json:"state"`
	GridSize         int               `json:"grid_size"`
	Players          []string          `json:"players"`
	CurrentTurn      int               `json:"current_turn"`
	CurrentAnnouncer string            `json:"current_announcer,omitempty"`
	CurrentLetter    *string           `json:"current_letter"`
	Placements       map[string]bool   `json:"placements,omitempty"`
	MyBoard          *Board            `json:"my_board,omitempty"`
	AllBoards        map[string]*Board `json:"all_boards,omitempty"`
	Scores           []BoardScore      `json:"scores,omitempty"`
	Winner           *string           `json:"winner,omitempty"`
}

// Board response type
type Board struct {
	Cells [][]string `json:"cells"`
}

// BoardScore response type
type BoardScore struct {
	PlayerID   string      `json:"player_id"`
	TotalScore int         `json:"total_score"`
	Words      []WordMatch `json:"words"`
}

// WordMatch response type
type WordMatch struct {
	Word       string `json:"word"`
	Score      int    `json:"score"`
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Horizontal bool   `json:"horizontal"`
}

// AnnounceResult response type
type AnnounceResult struct {
	State         string `json:"state"`
	CurrentLetter string `json:"current_letter"`
}

// PlaceResult response type
type PlaceResult struct {
	Placed        bool         `json:"placed"`
	Board         Board        `json:"board"`
	TurnComplete  bool         `json:"turn_complete"`
	GameComplete  bool         `json:"game_complete,omitempty"`
	NextAnnouncer string       `json:"next_announcer,omitempty"`
	Scores        []BoardScore `json:"scores,omitempty"`
	Winner        *string      `json:"winner,omitempty"`
}

// HealthResult response type
type HealthResult struct {
	Status string `json:"status"`
}

func (o *Output) printPlayer(p Player) {
	guestStr := "no"
	if p.IsGuest {
		guestStr = "yes"
	}
	fmt.Printf("Player: %s (%s)\n", p.DisplayName, p.ID)
	fmt.Printf("Guest: %s\n", guestStr)
}

func (o *Output) printAuthResult(a AuthResult) {
	o.printPlayer(a.Player)
	fmt.Printf("Token: %s\n", a.SessionToken)
}

func (o *Output) printLobby(l Lobby) {
	fmt.Printf("Lobby: %s\n", l.Code)
	fmt.Printf("State: %s\n", l.State)
	fmt.Printf("Grid Size: %d\n", l.Config.GridSize)
	if l.CurrentGame != nil {
		fmt.Printf("Current Game: %s\n", *l.CurrentGame)
	}
	fmt.Printf("Members (%d):\n", len(l.Members))
	for _, m := range l.Members {
		hostStr := ""
		if m.IsHost {
			hostStr = " [host]"
		}
		fmt.Printf("  - %s (%s) - %s%s\n", m.DisplayName, m.PlayerID, m.Role, hostStr)
	}
}

func (o *Output) printLobbyConfig(c LobbyConfig) {
	fmt.Printf("Grid Size: %d\n", c.GridSize)
}

func (o *Output) printGameState(g GameState) {
	fmt.Printf("Game: %s\n", g.ID)
	fmt.Printf("State: %s\n", g.State)
	fmt.Printf("Turn: %d\n", g.CurrentTurn)
	fmt.Printf("Grid Size: %d\n", g.GridSize)

	if g.CurrentAnnouncer != "" {
		fmt.Printf("Announcer: %s\n", g.CurrentAnnouncer)
	}
	if g.CurrentLetter != nil {
		fmt.Printf("Current Letter: %s\n", *g.CurrentLetter)
	}

	if len(g.Placements) > 0 {
		placed := []string{}
		waiting := []string{}
		for pid, p := range g.Placements {
			if p {
				placed = append(placed, pid)
			} else {
				waiting = append(waiting, pid)
			}
		}
		if len(placed) > 0 {
			fmt.Printf("Placed: %s\n", strings.Join(placed, ", "))
		}
		if len(waiting) > 0 {
			fmt.Printf("Waiting: %s\n", strings.Join(waiting, ", "))
		}
	}

	if g.MyBoard != nil {
		fmt.Println("\nYour Board:")
		o.printBoard(g.MyBoard)
	}

	if g.AllBoards != nil {
		for pid, board := range g.AllBoards {
			fmt.Printf("\nBoard (%s):\n", pid)
			o.printBoard(board)
		}
	}

	if len(g.Scores) > 0 {
		fmt.Println("\nScores:")
		for _, s := range g.Scores {
			fmt.Printf("  %s: %d points\n", s.PlayerID, s.TotalScore)
			for _, w := range s.Words {
				fmt.Printf("    - %s (%d pts)\n", w.Word, w.Score)
			}
		}
	}

	if g.Winner != nil {
		fmt.Printf("\nWinner: %s\n", *g.Winner)
	}
}

func (o *Output) printBoard(b *Board) {
	if b == nil || len(b.Cells) == 0 {
		return
	}

	size := len(b.Cells)

	// Print column headers
	fmt.Print("    ")
	for col := 0; col < size; col++ {
		fmt.Printf(" %d ", col)
	}
	fmt.Println()

	// Print top border
	fmt.Print("   +")
	for col := 0; col < size; col++ {
		fmt.Print("---")
	}
	fmt.Println("+")

	// Print rows
	for row := 0; row < size; row++ {
		fmt.Printf(" %d |", row)
		for col := 0; col < size; col++ {
			cell := b.Cells[row][col]
			if cell == "" {
				fmt.Print(" . ")
			} else {
				fmt.Printf(" %s ", cell)
			}
		}
		fmt.Println("|")
	}

	// Print bottom border
	fmt.Print("   +")
	for col := 0; col < size; col++ {
		fmt.Print("---")
	}
	fmt.Println("+")
}

func (o *Output) printAnnounceResult(a AnnounceResult) {
	fmt.Printf("Letter announced: %s\n", a.CurrentLetter)
	fmt.Printf("Game state: %s\n", a.State)
}

func (o *Output) printPlaceResult(p PlaceResult) {
	if p.Placed {
		fmt.Println("Letter placed successfully")
	}

	if p.TurnComplete {
		fmt.Println("Turn complete!")
		if p.NextAnnouncer != "" {
			fmt.Printf("Next announcer: %s\n", p.NextAnnouncer)
		}
	}

	if p.GameComplete {
		fmt.Println("Game complete!")
		if p.Winner != nil {
			fmt.Printf("Winner: %s\n", *p.Winner)
		}
		if len(p.Scores) > 0 {
			fmt.Println("\nFinal Scores:")
			for _, s := range p.Scores {
				fmt.Printf("  %s: %d points\n", s.PlayerID, s.TotalScore)
			}
		}
	}
}

func (o *Output) printHealthResult(h HealthResult) {
	fmt.Printf("Status: %s\n", h.Status)
}
