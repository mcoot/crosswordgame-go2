package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/pages"
)

// GameHandler handles game pages and actions
type GameHandler struct {
	lobbyController *lobby.Controller
	gameController  *game.Controller
	boardService    *board.Service
	hubManager      *sse.HubManager
	broadcaster     *sse.Broadcaster
}

// NewGameHandler creates a new GameHandler
func NewGameHandler(lobbyController *lobby.Controller, gameController *game.Controller, boardService *board.Service, hubManager *sse.HubManager) *GameHandler {
	return &GameHandler{
		lobbyController: lobbyController,
		gameController:  gameController,
		boardService:    boardService,
		hubManager:      hubManager,
		broadcaster:     sse.NewBroadcaster(hubManager),
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

	// Check player's role
	member := lob.GetMember(player.ID)
	isSpectator := member == nil || member.Role == model.RoleSpectator

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

	// For spectators, get all boards
	var allBoards map[model.PlayerID]*model.Board
	if isSpectator || g.State == model.GameStateScoring {
		boards, _ := h.boardService.GetBoardsForGame(r.Context(), g.ID)
		allBoards = make(map[model.PlayerID]*model.Board)
		for _, b := range boards {
			allBoards[b.PlayerID] = b
		}
	}

	flash := middleware.GetFlash(r.Context())

	data := pages.GameData{
		PageData: layout.PageData{
			Title:  "Game - " + string(lob.Code),
			Player: player,
			Flash:  flash,
		},
		Lobby:       lob,
		Game:        g,
		MyBoard:     myBoard,
		IsAnnouncer: isAnnouncer,
		HasPlaced:   hasPlaced,
		IsSpectator: isSpectator || !isInGame,
		AllBoards:   allBoards,
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

	http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
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
	} else {
		// Broadcast letter announced to all game clients
		g, _ := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
		if g != nil {
			h.broadcaster.BroadcastLetterAnnounced(r.Context(), g, code)
		}
	}

	http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
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
	} else {
		// Broadcast placement update
		g, _ := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
		if g != nil {
			h.broadcaster.BroadcastPlacementUpdate(r.Context(), g, code, player.ID)

			// Check if game is complete
			switch g.State {
			case model.GameStateScoring:
				h.broadcaster.BroadcastGameComplete(code)
			case model.GameStateAnnouncing:
				// All placed, new turn started - tell clients to refresh
				h.broadcaster.BroadcastTurnComplete(r.Context(), g, code)
			}
		}
	}

	http.Redirect(w, r, "/lobby/"+string(code)+"/game", http.StatusSeeOther)
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
	} else {
		middleware.SetFlash(w, "info", "Game abandoned")
		// Broadcast game-abandoned so all clients go back to lobby
		h.broadcaster.BroadcastGameAbandoned(code)
	}

	http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
}
