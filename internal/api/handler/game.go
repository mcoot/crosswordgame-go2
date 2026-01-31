package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/api/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/api/request"
	"github.com/mcoot/crosswordgame-go2/internal/api/response"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// GameHandler handles game-related endpoints
type GameHandler struct {
	lobbyController *lobby.Controller
	gameController  *game.Controller
	boardService    *board.Service
	botService      *bot.Service
	hubManager      *sse.HubManager
	broadcaster     *sse.Broadcaster
}

// NewGameHandler creates a new game handler
func NewGameHandler(
	lobbyController *lobby.Controller,
	gameController *game.Controller,
	boardService *board.Service,
	botService *bot.Service,
	hubManager *sse.HubManager,
	logger *slog.Logger,
) *GameHandler {
	var broadcaster *sse.Broadcaster
	if hubManager != nil {
		broadcaster = sse.NewBroadcaster(hubManager, logger)
	}
	return &GameHandler{
		lobbyController: lobbyController,
		gameController:  gameController,
		boardService:    boardService,
		botService:      botService,
		hubManager:      hubManager,
		broadcaster:     broadcaster,
	}
}

// getBroadcaster returns the broadcaster if available
func (h *GameHandler) getBroadcaster() *sse.Broadcaster {
	return h.broadcaster
}

// Start handles POST /api/v1/lobbies/{code}/game
func (h *GameHandler) Start(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	g, err := h.lobbyController.StartGame(r.Context(), code, player.ID)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast game started to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastGameStarted(code)
	}

	// Process bot actions after game start
	h.processBotActions(r.Context(), g.ID, code)

	resp := response.GameStateFromModel(g, nil, nil, nil, "")
	response.JSON(w, http.StatusCreated, resp)
}

// Get handles GET /api/v1/lobbies/{code}/game
func (h *GameHandler) Get(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	if lob.CurrentGame == nil {
		WriteError(w, model.ErrNoGameInProgress)
		return
	}

	g, err := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Determine what boards to show based on role and game state
	member := lob.GetMember(player.ID)
	isSpectator := member != nil && member.Role == model.RoleSpectator
	isGameComplete := g.State == model.GameStateScoring

	var myBoard *model.Board
	var allBoards map[model.PlayerID]*model.Board
	var scores []model.BoardScore
	var winner model.PlayerID

	if isSpectator || isGameComplete {
		// Show all boards
		boards, err := h.boardService.GetBoardsForGame(r.Context(), g.ID)
		if err != nil {
			WriteError(w, err)
			return
		}
		allBoards = make(map[model.PlayerID]*model.Board)
		for _, b := range boards {
			allBoards[b.PlayerID] = b
		}
	} else {
		// Show only player's own board
		myBoard, err = h.boardService.GetBoard(r.Context(), g.ID, player.ID)
		if err != nil && !errors.Is(err, model.ErrBoardNotFound) {
			WriteError(w, err)
			return
		}
	}

	// Include scores if game is complete
	if isGameComplete {
		scores, err = h.gameController.GetFinalScores(r.Context(), g.ID)
		if err != nil {
			WriteError(w, err)
			return
		}
		summary, err := h.gameController.CreateGameSummary(r.Context(), g.ID)
		if err == nil {
			winner = summary.Winner
		}
	}

	resp := response.GameStateFromModel(g, myBoard, allBoards, scores, winner)
	response.JSON(w, http.StatusOK, resp)
}

// Announce handles POST /api/v1/lobbies/{code}/game/announce
func (h *GameHandler) Announce(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	var req request.AnnounceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	if len(req.Letter) != 1 {
		WriteError(w, NewInvalidRequestError("letter must be a single character"))
		return
	}

	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	if lob.CurrentGame == nil {
		WriteError(w, model.ErrNoGameInProgress)
		return
	}

	letter := rune(req.Letter[0])
	if err := h.gameController.AnnounceLetter(r.Context(), *lob.CurrentGame, player.ID, letter); err != nil {
		WriteError(w, err)
		return
	}

	g, err := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast letter announced to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastLetterAnnounced(r.Context(), g, code)
	}

	// Process bot actions after announcement
	h.processBotActions(r.Context(), *lob.CurrentGame, code)

	resp := response.AnnounceResponse{
		State:         string(g.State),
		CurrentLetter: string(g.CurrentLetter),
	}
	response.JSON(w, http.StatusOK, resp)
}

// Place handles POST /api/v1/lobbies/{code}/game/place
func (h *GameHandler) Place(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	var req request.PlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	if lob.CurrentGame == nil {
		WriteError(w, model.ErrNoGameInProgress)
		return
	}

	pos := model.Position{Row: req.Row, Col: req.Col}
	if err := h.gameController.PlaceLetter(r.Context(), *lob.CurrentGame, player.ID, pos); err != nil {
		WriteError(w, err)
		return
	}

	// Get updated game state
	g, err := h.gameController.GetGame(r.Context(), *lob.CurrentGame)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Get player's board
	boardObj, err := h.boardService.GetBoard(r.Context(), g.ID, player.ID)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp := response.PlaceResponse{
		Placed:       true,
		Board:        response.BoardFromModel(boardObj),
		TurnComplete: g.State == model.GameStateAnnouncing || g.State == model.GameStateScoring,
		GameComplete: g.State == model.GameStateScoring,
	}

	// Broadcast placement update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastPlacementUpdate(r.Context(), g, code, player.ID)

		// Broadcast turn or game completion
		switch g.State {
		case model.GameStateAnnouncing:
			b.BroadcastTurnComplete(r.Context(), g, code)
		case model.GameStateScoring:
			b.BroadcastGameComplete(code)
		}
	}

	// If turn advanced, include next announcer
	if g.State == model.GameStateAnnouncing {
		resp.NextAnnouncer = string(g.CurrentAnnouncer())
	}

	// If game complete, include scores and handle lobby state update
	if g.State == model.GameStateScoring {
		scores, err := h.gameController.GetFinalScores(r.Context(), g.ID)
		if err == nil {
			resp.Scores = make([]response.BoardScore, len(scores))
			for i, s := range scores {
				resp.Scores[i] = response.BoardScoreFromModel(s)
			}
		}

		summary, err := h.gameController.CreateGameSummary(r.Context(), g.ID)
		if err == nil && summary.Winner != "" {
			w := string(summary.Winner)
			resp.Winner = &w
		}

		// Complete the game in the lobby
		_ = h.lobbyController.CompleteGame(r.Context(), code)
	}

	// Process bot actions after placement (only if game still active)
	if g.State != model.GameStateScoring && g.State != model.GameStateAbandoned {
		h.processBotActions(r.Context(), g.ID, code)
	}

	response.JSON(w, http.StatusOK, resp)
}

// processBotActions runs bot actions and broadcasts SSE updates
func (h *GameHandler) processBotActions(ctx context.Context, gameID model.GameID, code model.LobbyCode) {
	if h.botService == nil {
		return
	}

	actions, err := h.botService.ProcessBotActions(ctx, gameID)
	if err != nil || len(actions) == 0 {
		return
	}

	h.broadcastBotActions(ctx, actions, code, gameID)
}

// broadcastBotActions sends SSE broadcasts for bot actions
func (h *GameHandler) broadcastBotActions(ctx context.Context, actions []bot.BotAction, code model.LobbyCode, gameID model.GameID) {
	b := h.getBroadcaster()
	if b == nil {
		return
	}

	for _, action := range actions {
		switch action.Type {
		case bot.ActionAnnounce:
			g, err := h.gameController.GetGame(ctx, gameID)
			if err == nil {
				b.BroadcastLetterAnnounced(ctx, g, code)
			}
		case bot.ActionPlace:
			g, err := h.gameController.GetGame(ctx, gameID)
			if err == nil {
				b.BroadcastPlacementUpdate(ctx, g, code, action.PlayerID)
			}
		case bot.ActionTurnComplete:
			g, err := h.gameController.GetGame(ctx, gameID)
			if err == nil {
				b.BroadcastTurnComplete(ctx, g, code)
			}
		case bot.ActionGameComplete:
			b.BroadcastGameComplete(code)
			// Complete the game in the lobby
			_ = h.lobbyController.CompleteGame(ctx, code)
		}
	}
}

// Abandon handles DELETE /api/v1/lobbies/{code}/game
func (h *GameHandler) Abandon(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	if err := h.lobbyController.AbandonGame(r.Context(), code, player.ID); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast game abandoned to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastGameAbandoned(code)
	}

	response.NoContent(w)
}
