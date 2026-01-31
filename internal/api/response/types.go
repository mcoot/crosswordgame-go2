package response

import (
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
)

// Player represents a player in API responses
type Player struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	IsGuest     bool   `json:"is_guest"`
	IsBot       bool   `json:"is_bot,omitempty"`
}

// PlayerFromModel converts a model.Player to a response Player
func PlayerFromModel(p *model.Player) Player {
	return Player{
		ID:          string(p.ID),
		DisplayName: p.DisplayName,
		IsGuest:     p.IsGuest,
		IsBot:       p.IsBot,
	}
}

// AuthResponse is the response for authentication endpoints
type AuthResponse struct {
	Player       Player `json:"player"`
	SessionToken string `json:"session_token"`
}

// AuthResponseFromSession creates an AuthResponse from a session
func AuthResponseFromSession(s *auth.Session) AuthResponse {
	return AuthResponse{
		Player:       PlayerFromModel(&s.Player),
		SessionToken: s.Token,
	}
}

// LobbyConfig represents lobby configuration
type LobbyConfig struct {
	GridSize int `json:"grid_size"`
}

// LobbyConfigFromModel converts model.LobbyConfig
func LobbyConfigFromModel(c model.LobbyConfig) LobbyConfig {
	return LobbyConfig{
		GridSize: c.GridSize,
	}
}

// LobbyMember represents a lobby member
type LobbyMember struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsHost      bool   `json:"is_host"`
	IsBot       bool   `json:"is_bot,omitempty"`
}

// LobbyMemberFromModel converts model.LobbyMember
func LobbyMemberFromModel(m model.LobbyMember) LobbyMember {
	return LobbyMember{
		PlayerID:    string(m.Player.ID),
		DisplayName: m.Player.DisplayName,
		Role:        string(m.Role),
		IsHost:      m.IsHost,
		IsBot:       m.Player.IsBot,
	}
}

// GameSummary represents a completed game summary
type GameSummary struct {
	ID          string         `json:"id"`
	FinalScores map[string]int `json:"final_scores"`
	Winner      *string        `json:"winner"`
	CompletedAt time.Time      `json:"completed_at"`
}

// GameSummaryFromModel converts model.GameSummary
func GameSummaryFromModel(g model.GameSummary) GameSummary {
	scores := make(map[string]int, len(g.FinalScores))
	for pid, score := range g.FinalScores {
		scores[string(pid)] = score
	}
	var winner *string
	if g.Winner != "" {
		w := string(g.Winner)
		winner = &w
	}
	return GameSummary{
		ID:          string(g.ID),
		FinalScores: scores,
		Winner:      winner,
		CompletedAt: g.CompletedAt,
	}
}

// Lobby represents a lobby in API responses
type Lobby struct {
	Code        string        `json:"code"`
	State       string        `json:"state"`
	Config      LobbyConfig   `json:"config"`
	Members     []LobbyMember `json:"members"`
	CurrentGame *string       `json:"current_game"`
	GameHistory []GameSummary `json:"game_history,omitempty"`
}

// LobbyFromModel converts model.Lobby
func LobbyFromModel(l *model.Lobby) Lobby {
	members := make([]LobbyMember, len(l.Members))
	for i, m := range l.Members {
		members[i] = LobbyMemberFromModel(m)
	}

	history := make([]GameSummary, len(l.GameHistory))
	for i, g := range l.GameHistory {
		history[i] = GameSummaryFromModel(g)
	}

	var currentGame *string
	if l.CurrentGame != nil {
		g := string(*l.CurrentGame)
		currentGame = &g
	}

	return Lobby{
		Code:        string(l.Code),
		State:       string(l.State),
		Config:      LobbyConfigFromModel(l.Config),
		Members:     members,
		CurrentGame: currentGame,
		GameHistory: history,
	}
}

// Board represents a game board
type Board struct {
	Cells [][]string `json:"cells"`
}

// BoardFromModel converts model.Board to response Board
// Empty cells are represented as empty strings
func BoardFromModel(b *model.Board) Board {
	cells := make([][]string, b.Size)
	for row := 0; row < b.Size; row++ {
		cells[row] = make([]string, b.Size)
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] != 0 {
				cells[row][col] = string(b.Cells[row][col])
			}
		}
	}
	return Board{Cells: cells}
}

// WordMatch represents a word found on a board
type WordMatch struct {
	Word       string `json:"word"`
	Score      int    `json:"score"`
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Horizontal bool   `json:"horizontal"`
}

// WordMatchFromModel converts model.WordMatch
func WordMatchFromModel(w model.WordMatch) WordMatch {
	return WordMatch{
		Word:       w.Word,
		Score:      w.Score,
		Row:        w.StartPos.Row,
		Col:        w.StartPos.Col,
		Horizontal: w.Horizontal,
	}
}

// BoardScore represents a player's score
type BoardScore struct {
	PlayerID   string      `json:"player_id"`
	TotalScore int         `json:"total_score"`
	Words      []WordMatch `json:"words"`
}

// BoardScoreFromModel converts model.BoardScore
func BoardScoreFromModel(s model.BoardScore) BoardScore {
	words := make([]WordMatch, len(s.Words))
	for i, w := range s.Words {
		words[i] = WordMatchFromModel(w)
	}
	return BoardScore{
		PlayerID:   string(s.PlayerID),
		TotalScore: s.TotalScore,
		Words:      words,
	}
}

// GameState represents the current game state
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

// GameStateFromModel converts model.Game to response GameState
func GameStateFromModel(g *model.Game, myBoard *model.Board, allBoards map[model.PlayerID]*model.Board, scores []model.BoardScore, winner model.PlayerID) GameState {
	players := make([]string, len(g.Players))
	for i, p := range g.Players {
		players[i] = string(p)
	}

	placements := make(map[string]bool, len(g.Placements))
	for pid, placed := range g.Placements {
		placements[string(pid)] = placed
	}

	var currentLetter *string
	if g.CurrentLetter != 0 {
		l := string(g.CurrentLetter)
		currentLetter = &l
	}

	var myBoardResp *Board
	if myBoard != nil {
		b := BoardFromModel(myBoard)
		myBoardResp = &b
	}

	var allBoardsResp map[string]*Board
	if allBoards != nil {
		allBoardsResp = make(map[string]*Board, len(allBoards))
		for pid, board := range allBoards {
			b := BoardFromModel(board)
			allBoardsResp[string(pid)] = &b
		}
	}

	var scoresResp []BoardScore
	if scores != nil {
		scoresResp = make([]BoardScore, len(scores))
		for i, s := range scores {
			scoresResp[i] = BoardScoreFromModel(s)
		}
	}

	var winnerResp *string
	if winner != "" {
		w := string(winner)
		winnerResp = &w
	}

	return GameState{
		ID:               string(g.ID),
		State:            string(g.State),
		GridSize:         g.GridSize,
		Players:          players,
		CurrentTurn:      g.CurrentTurn,
		CurrentAnnouncer: string(g.CurrentAnnouncer()),
		CurrentLetter:    currentLetter,
		Placements:       placements,
		MyBoard:          myBoardResp,
		AllBoards:        allBoardsResp,
		Scores:           scoresResp,
		Winner:           winnerResp,
	}
}

// AnnounceResponse is the response after announcing a letter
type AnnounceResponse struct {
	State         string `json:"state"`
	CurrentLetter string `json:"current_letter"`
}

// PlaceResponse is the response after placing a letter
type PlaceResponse struct {
	Placed        bool         `json:"placed"`
	Board         Board        `json:"board"`
	TurnComplete  bool         `json:"turn_complete"`
	GameComplete  bool         `json:"game_complete,omitempty"`
	NextAnnouncer string       `json:"next_announcer,omitempty"`
	Scores        []BoardScore `json:"scores,omitempty"`
	Winner        *string      `json:"winner,omitempty"`
}
