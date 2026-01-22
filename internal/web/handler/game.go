package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/components"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/pages"
)

// GameHandler handles game pages and actions
type GameHandler struct {
	lobbyController *lobby.Controller
	gameController  *game.Controller
	boardService    *board.Service
	scoringService  *scoring.Service
	hubManager      *sse.HubManager
	broadcaster     *sse.Broadcaster
}

// NewGameHandler creates a new GameHandler
func NewGameHandler(lobbyController *lobby.Controller, gameController *game.Controller, boardService *board.Service, scoringService *scoring.Service, hubManager *sse.HubManager, logger *slog.Logger) *GameHandler {
	return &GameHandler{
		lobbyController: lobbyController,
		gameController:  gameController,
		boardService:    boardService,
		scoringService:  scoringService,
		hubManager:      hubManager,
		broadcaster:     sse.NewBroadcaster(hubManager, logger),
	}
}

// View renders the game page
func (h *GameHandler) View(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		middleware.SetFlash(w, "error", "Lobby not found")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Check if there's an active game
	if lob.CurrentGame == nil {
		middleware.SetFlash(w, "info", "No game in progress")
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	g, err := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	if err != nil {
		middleware.SetFlash(w, "error", "Game not found")
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	// Check player's role and host status
	member := lob.GetMember(player.ID)
	isSpectator := member == nil || member.Role == model.RoleSpectator
	host := lob.GetHost()
	isHost := host != nil && host.Player.ID == player.ID

	// Check if player is in the game
	isInGame := false
	for _, pid := range g.Players {
		if pid == player.ID {
			isInGame = true
			break
		}
	}

	// Get player's board (if they're a player)
	var myBoard *model.Board
	if isInGame {
		myBoard, _ = h.boardService.GetBoard(r.Context(), g.ID, player.ID)
	}

	// Check if current player is the announcer
	isAnnouncer := g.CurrentAnnouncer() == player.ID

	// Check if player has placed this turn
	hasPlaced := g.Placements[player.ID]

	// For spectators or scoring, get all boards
	var allBoards map[model.PlayerID]*model.Board
	var boardsList []*model.Board
	if isSpectator || g.State == model.GameStateScoring {
		boardsList, _ = h.boardService.GetBoardsForGame(r.Context(), g.ID)
		allBoards = make(map[model.PlayerID]*model.Board)
		for _, b := range boardsList {
			allBoards[b.PlayerID] = b
		}
	}

	// Calculate scores if game is complete
	var scores []model.BoardScore
	var winner model.PlayerID
	if g.State == model.GameStateScoring && len(boardsList) > 0 {
		scores = h.scoringService.ScoreMultipleBoards(boardsList)
		winner = h.scoringService.DetermineWinner(scores)
	}

	// Build player names map from lobby members
	playerNames := make(map[model.PlayerID]string)
	for _, m := range lob.Members {
		playerNames[m.Player.ID] = m.Player.DisplayName
	}

	flash := middleware.GetFlash(r.Context())
	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())

	data := pages.GameData{
		PageData: layout.PageData{
			Title:           "Game - " + string(lob.Code),
			Player:          player,
			Flash:           flash,
			ActiveLobbyCode: activeLobbyCode,
		},
		Lobby:       lob,
		Game:        g,
		MyBoard:     myBoard,
		IsAnnouncer: isAnnouncer,
		HasPlaced:   hasPlaced,
		IsSpectator: isSpectator || !isInGame,
		IsHost:      isHost,
		AllBoards:   allBoards,
		Scores:      scores,
		Winner:      winner,
		PlayerNames: playerNames,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Game(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Start handles starting a new game
func (h *GameHandler) Start(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	_, err := h.lobbyController.StartGame(r.Context(), code, player.ID)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not start game: "+err.Error())
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	// Broadcast game started to all lobby clients
	h.broadcaster.BroadcastGameStarted(code)

	// Use HX-Redirect for HTMX-aware client-side navigation
	w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
	w.WriteHeader(http.StatusNoContent)
}

// Announce handles letter announcement
func (h *GameHandler) Announce(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "error", "Invalid form data")
		http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
		return
	}

	letterStr := strings.ToUpper(strings.TrimSpace(r.FormValue("letter")))
	if len(letterStr) != 1 {
		middleware.SetFlash(w, "error", "Please select a letter")
		http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
		return
	}
	letter := rune(letterStr[0])

	// Get the current game
	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil || lob.CurrentGame == nil {
		middleware.SetFlash(w, "error", "No game in progress")
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	err = h.gameController.AnnounceLetter(r.Context(), *lob.CurrentGame, player.ID, letter)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not announce letter: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast letter announced to all game clients
	g, _ := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	if g != nil {
		h.broadcaster.BroadcastLetterAnnounced(r.Context(), g, code)
	}

	// SSE broadcast handles the UI update, so just return 204
	w.WriteHeader(http.StatusNoContent)
}

// Place handles letter placement
func (h *GameHandler) Place(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "error", "Invalid form data")
		http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
		return
	}

	row, err := strconv.Atoi(r.FormValue("row"))
	if err != nil {
		middleware.SetFlash(w, "error", "Invalid row")
		http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
		return
	}

	col, err := strconv.Atoi(r.FormValue("col"))
	if err != nil {
		middleware.SetFlash(w, "error", "Invalid column")
		http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
		return
	}

	// Get the current game
	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil || lob.CurrentGame == nil {
		middleware.SetFlash(w, "error", "No game in progress")
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	pos := model.Position{Row: row, Col: col}
	err = h.gameController.PlaceLetter(r.Context(), *lob.CurrentGame, player.ID, pos)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not place letter: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Get updated game and board state
	g, _ := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	board, _ := h.boardService.GetBoard(r.Context(), *lob.CurrentGame, player.ID)

	if g != nil {
		// Broadcast placement count to other players via SSE
		h.broadcaster.BroadcastPlacementUpdate(r.Context(), g, code, player.ID)

		// Check if game advanced state
		switch g.State {
		case model.GameStateScoring:
			h.broadcaster.BroadcastGameComplete(code)
		case model.GameStateAnnouncing:
			// All placed, new turn started - tell clients to refresh
			h.broadcaster.BroadcastTurnComplete(r.Context(), g, code)
		}
	}

	// Return OOB swaps to update the placing player's UI immediately
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var buf bytes.Buffer

	// 1. Updated game board (shows placed letter, disables remaining cells)
	buf.WriteString(`<div id="game-board" hx-swap-oob="true">`)
	_ = components.GameBoard(code, board, g, true).Render(r.Context(), &buf)
	buf.WriteString(`</div>`)

	// 2. Updated game status ("Waiting for other players...")
	buf.WriteString(`<div id="game-status" hx-swap-oob="true">`)
	announcerName := getPlayerName(lob, g.CurrentAnnouncer())
	_ = components.GameStatus(g, false, true, announcerName).Render(r.Context(), &buf)
	buf.WriteString(`</div>`)

	// 3. Updated placement count
	if g != nil && g.State == model.GameStatePlacing {
		placedCount := countPlacements(g)
		buf.WriteString(`<div id="placement-status" hx-swap-oob="true" class="text-muted">`)
		buf.WriteString(strconv.Itoa(placedCount) + "/" + strconv.Itoa(len(g.Players)) + " players have placed")
		buf.WriteString(`</div>`)
	}

	_, _ = w.Write(buf.Bytes())
}

// Abandon handles game abandonment
func (h *GameHandler) Abandon(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	err := h.lobbyController.AbandonGame(r.Context(), code, player.ID)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not abandon game: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	middleware.SetFlash(w, "info", "Game abandoned")
	// Broadcast game-abandoned so all clients go back to lobby
	h.broadcaster.BroadcastGameAbandoned(code)

	// Use HX-Redirect for HTMX-aware client-side navigation to lobby
	w.Header().Set("HX-Redirect", "/lobby/"+string(code))
	w.WriteHeader(http.StatusNoContent)
}

// Dismiss handles dismissing game scores and returning to lobby
func (h *GameHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	// Parse form to check for start_new flag
	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "error", "Invalid form data")
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	startNew := r.FormValue("start_new") == "true"

	// Verify player is host
	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		middleware.SetFlash(w, "error", "Lobby not found")
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	host := lob.GetHost()
	if host == nil || host.Player.ID != player.ID {
		middleware.SetFlash(w, "error", "Only the host can dismiss the game")
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Complete the game (saves summary to history and returns lobby to waiting state)
	err = h.lobbyController.CompleteGame(r.Context(), code)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not dismiss game: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// If start_new flag is set, start a new game immediately
	if startNew {
		_, err = h.lobbyController.StartGame(r.Context(), code, player.ID)
		if err != nil {
			middleware.SetFlash(w, "error", "Could not start new game: "+err.Error())
			h.broadcaster.BroadcastGameDismissed(code)
			w.Header().Set("HX-Redirect", "/lobby/"+string(code))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Broadcast game started so all clients go to game page
		h.broadcaster.BroadcastGameStarted(code)
		w.Header().Set("HX-Redirect", "/lobby/"+string(code)+"/game")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast game-dismissed so all clients go back to lobby
	h.broadcaster.BroadcastGameDismissed(code)

	// Use HX-Redirect for HTMX-aware client-side navigation to lobby
	w.Header().Set("HX-Redirect", "/lobby/"+string(code))
	w.WriteHeader(http.StatusNoContent)
}

// countPlacements counts how many players have placed in the current turn
func countPlacements(g *model.Game) int {
	count := 0
	for _, placed := range g.Placements {
		if placed {
			count++
		}
	}
	return count
}

// getPlayerName returns the display name for a player ID from the lobby members
func getPlayerName(lob *model.Lobby, playerID model.PlayerID) string {
	for _, m := range lob.Members {
		if m.Player.ID == playerID {
			return m.Player.DisplayName
		}
	}
	return ""
}
