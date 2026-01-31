package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/api/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/api/request"
	"github.com/mcoot/crosswordgame-go2/internal/api/response"
	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// LobbyHandler handles lobby-related endpoints
type LobbyHandler struct {
	lobbyController *lobby.Controller
	botService      *bot.Service
	hubManager      *sse.HubManager
	broadcaster     *sse.Broadcaster
}

// NewLobbyHandler creates a new lobby handler
func NewLobbyHandler(lobbyController *lobby.Controller, botService *bot.Service, hubManager *sse.HubManager, logger *slog.Logger) *LobbyHandler {
	var broadcaster *sse.Broadcaster
	if hubManager != nil {
		broadcaster = sse.NewBroadcaster(hubManager, logger)
	}
	return &LobbyHandler{
		lobbyController: lobbyController,
		botService:      botService,
		hubManager:      hubManager,
		broadcaster:     broadcaster,
	}
}

// getBroadcaster returns the broadcaster if available
func (h *LobbyHandler) getBroadcaster() *sse.Broadcaster {
	return h.broadcaster
}

// Create handles POST /api/v1/lobbies
func (h *LobbyHandler) Create(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())

	var req request.CreateLobbyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for default config
		req = request.CreateLobbyRequest{}
	}

	lobby, err := h.lobbyController.CreateLobby(r.Context(), *player)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Update config if grid size provided
	if req.GridSize > 0 {
		config := model.LobbyConfig{GridSize: req.GridSize}
		if err := h.lobbyController.UpdateConfig(r.Context(), lobby.Code, player.ID, config); err != nil {
			WriteError(w, err)
			return
		}
		lobby.Config = config
	}

	response.JSON(w, http.StatusCreated, response.LobbyFromModel(lobby))
}

// Get handles GET /api/v1/lobbies/{code}
func (h *LobbyHandler) Get(w http.ResponseWriter, r *http.Request) {
	code := model.LobbyCode(mux.Vars(r)["code"])

	lobby, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, response.LobbyFromModel(lobby))
}

// Join handles POST /api/v1/lobbies/{code}/join
func (h *LobbyHandler) Join(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	if err := h.lobbyController.JoinLobby(r.Context(), code, *player); err != nil {
		WriteError(w, err)
		return
	}

	lobby, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast member list update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastMemberListUpdate(r.Context(), lobby)
	}

	response.JSON(w, http.StatusOK, response.LobbyFromModel(lobby))
}

// Leave handles POST /api/v1/lobbies/{code}/leave
func (h *LobbyHandler) Leave(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	if err := h.lobbyController.LeaveLobby(r.Context(), code, player.ID); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast member list update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		lobby, _ := h.lobbyController.GetLobby(r.Context(), code)
		if lobby != nil {
			b.BroadcastMemberListUpdate(r.Context(), lobby)
		}
	}

	response.NoContent(w)
}

// UpdateConfig handles PATCH /api/v1/lobbies/{code}/config
func (h *LobbyHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	var req request.UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	config := model.LobbyConfig{GridSize: req.GridSize}
	if err := h.lobbyController.UpdateConfig(r.Context(), code, player.ID, config); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast refresh to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastRefresh(code)
	}

	response.JSON(w, http.StatusOK, response.LobbyConfigFromModel(config))
}

// SetRole handles PATCH /api/v1/lobbies/{code}/members/{player_id}/role
func (h *LobbyHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	requestingPlayer := middleware.MustGetPlayer(r.Context())
	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])
	targetPlayerID := model.PlayerID(vars["player_id"])

	var req request.SetRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	// Verify requesting player is host
	lobby, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	host := lobby.GetHost()
	if host == nil || host.Player.ID != requestingPlayer.ID {
		WriteError(w, model.ErrNotHost)
		return
	}

	role := model.LobbyMemberRole(req.Role)
	if err := h.lobbyController.SetRole(r.Context(), code, targetPlayerID, role); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast member list update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		lobby, _ = h.lobbyController.GetLobby(r.Context(), code)
		if lobby != nil {
			b.BroadcastMemberListUpdate(r.Context(), lobby)
		}
	}

	response.NoContent(w)
}

// TransferHost handles POST /api/v1/lobbies/{code}/transfer-host
func (h *LobbyHandler) TransferHost(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	var req request.TransferHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	if req.NewHostID == "" {
		WriteError(w, NewInvalidRequestError("new_host_id is required"))
		return
	}

	newHostID := model.PlayerID(req.NewHostID)
	if err := h.lobbyController.TransferHost(r.Context(), code, player.ID, newHostID); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast refresh to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastRefresh(code)
	}

	response.NoContent(w)
}

// AddBot handles POST /api/v1/lobbies/{code}/bots
func (h *LobbyHandler) AddBot(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	code := model.LobbyCode(mux.Vars(r)["code"])

	var req request.AddBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = request.AddBotRequest{}
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = model.BotStrategyRandom
	}

	botPlayer, err := h.botService.AddBotToLobby(r.Context(), code, player.ID, strategy)
	if err != nil {
		WriteError(w, err)
		return
	}

	lobby, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast member list update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		b.BroadcastMemberListUpdate(r.Context(), lobby)
	}

	_ = botPlayer // included in lobby response
	response.JSON(w, http.StatusCreated, response.LobbyFromModel(lobby))
}

// RemoveBot handles DELETE /api/v1/lobbies/{code}/bots/{player_id}
func (h *LobbyHandler) RemoveBot(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])
	botPlayerID := model.PlayerID(vars["player_id"])

	if err := h.botService.RemoveBotFromLobby(r.Context(), code, player.ID, botPlayerID); err != nil {
		WriteError(w, err)
		return
	}

	// Broadcast member list update to SSE clients
	if b := h.getBroadcaster(); b != nil {
		lobby, _ := h.lobbyController.GetLobby(r.Context(), code)
		if lobby != nil {
			b.BroadcastMemberListUpdate(r.Context(), lobby)
		}
	}

	response.NoContent(w)
}
