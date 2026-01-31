package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/services/bot"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/pages"
)

// LobbyHandler handles lobby pages and actions
type LobbyHandler struct {
	lobbyController *lobby.Controller
	authService     *auth.Service
	botService      *bot.Service
	hubManager      *sse.HubManager
	broadcaster     *sse.Broadcaster
}

// NewLobbyHandler creates a new LobbyHandler
func NewLobbyHandler(lobbyController *lobby.Controller, authService *auth.Service, botService *bot.Service, hubManager *sse.HubManager, logger *slog.Logger) *LobbyHandler {
	return &LobbyHandler{
		lobbyController: lobbyController,
		authService:     authService,
		botService:      botService,
		hubManager:      hubManager,
		broadcaster:     sse.NewBroadcaster(hubManager, logger),
	}
}

// Create handles lobby creation
func (h *LobbyHandler) Create(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "error", "Invalid form data")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	gridSize := 5 // default
	if gs := r.FormValue("grid_size"); gs != "" {
		if parsed, err := strconv.Atoi(gs); err == nil && parsed >= 2 && parsed <= 7 {
			gridSize = parsed
		}
	}

	lob, err := h.lobbyController.CreateLobby(r.Context(), *player)
	if err != nil {
		middleware.SetFlash(w, "error", "Failed to create lobby")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Update config with grid size (non-fatal if it fails, continue with default config)
	cfg := model.LobbyConfig{GridSize: gridSize}
	_ = h.lobbyController.UpdateConfig(r.Context(), lob.Code, player.ID, cfg)

	middleware.SetFlash(w, "success", "Lobby created!")
	http.Redirect(w, r, "/lobby/"+string(lob.Code), http.StatusSeeOther)
}

// JoinByForm handles joining a lobby via form submission
func (h *LobbyHandler) JoinByForm(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		middleware.SetFlash(w, "error", "Invalid form data")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	if code == "" {
		middleware.SetFlash(w, "error", "Lobby code is required")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lobbyCode := model.LobbyCode(code)
	err := h.lobbyController.JoinLobby(r.Context(), lobbyCode, *player)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not join lobby: "+err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Broadcast member list update to existing clients
	lob, _ := h.lobbyController.GetLobby(r.Context(), lobbyCode)
	if lob != nil {
		h.broadcaster.BroadcastMemberListUpdate(r.Context(), lob)
	}

	middleware.SetFlash(w, "success", "Joined lobby!")
	http.Redirect(w, r, "/lobby/"+code, http.StatusSeeOther)
}

// View renders the lobby page
func (h *LobbyHandler) View(w http.ResponseWriter, r *http.Request) {
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

	// Check if player is a member
	member := lob.GetMember(player.ID)
	if member == nil {
		// Try to join the lobby
		if err := h.lobbyController.JoinLobby(r.Context(), code, *player); err != nil {
			middleware.SetFlash(w, "error", "Could not join lobby: "+err.Error())
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// Refresh lobby to get updated member list
		lob, err = h.lobbyController.GetLobby(r.Context(), code)
		if err != nil {
			middleware.SetFlash(w, "error", "Lobby not found")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		member = lob.GetMember(player.ID)

		// Broadcast member list update
		h.broadcaster.BroadcastMemberListUpdate(r.Context(), lob)
	}

	flash := middleware.GetFlash(r.Context())
	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())

	// Check if player is host
	host := lob.GetHost()
	isHost := host != nil && host.Player.ID == player.ID

	data := pages.LobbyData{
		PageData: layout.PageData{
			Title:           "Lobby " + string(lob.Code),
			Player:          player,
			Flash:           flash,
			ActiveLobbyCode: activeLobbyCode,
		},
		Lobby:    lob,
		IsHost:   isHost,
		MyRole:   member.Role,
		MyMember: member,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Lobby(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Leave handles leaving a lobby
func (h *LobbyHandler) Leave(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := h.lobbyController.LeaveLobby(r.Context(), code, player.ID); err != nil {
		middleware.SetFlash(w, "error", "Could not leave lobby: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast member list update to remaining clients
	lob, _ := h.lobbyController.GetLobby(r.Context(), code)
	if lob != nil {
		h.broadcaster.BroadcastMemberListUpdate(r.Context(), lob)
	}

	middleware.SetFlash(w, "info", "You left the lobby")
	// Use HX-Redirect for HTMX-aware client-side navigation
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusNoContent)
}

// UpdateConfig handles lobby config updates
func (h *LobbyHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	gridSize := 5
	if gs := r.FormValue("grid_size"); gs != "" {
		if parsed, err := strconv.Atoi(gs); err == nil && parsed >= 2 && parsed <= 7 {
			gridSize = parsed
		}
	}

	cfg := model.LobbyConfig{GridSize: gridSize}
	err := h.lobbyController.UpdateConfig(r.Context(), code, player.ID, cfg)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not update config: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	middleware.SetFlash(w, "success", "Settings updated")
	// Broadcast refresh so other clients see updated config
	h.broadcaster.BroadcastRefresh(code)

	// SSE broadcast handles the UI update, so just return 204
	w.WriteHeader(http.StatusNoContent)
}

// SetRole handles role changes
func (h *LobbyHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	targetPlayerID := model.PlayerID(r.FormValue("player_id"))
	roleStr := r.FormValue("role")

	var role model.LobbyMemberRole
	switch roleStr {
	case "player":
		role = model.RolePlayer
	case "spectator":
		role = model.RoleSpectator
	default:
		middleware.SetFlash(w, "error", "Invalid role")
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := h.lobbyController.SetRole(r.Context(), code, targetPlayerID, role)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not change role: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast refresh so all clients get personalized views with correct role buttons
	h.broadcaster.BroadcastRefresh(code)

	// SSE broadcast handles the UI update, so just return 204
	w.WriteHeader(http.StatusNoContent)
}

// TransferHost handles host transfer
func (h *LobbyHandler) TransferHost(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	newHostID := model.PlayerID(r.FormValue("new_host_id"))

	err := h.lobbyController.TransferHost(r.Context(), code, player.ID, newHostID)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not transfer host: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	middleware.SetFlash(w, "success", "Host transferred")
	// Broadcast refresh so all clients see updated host
	h.broadcaster.BroadcastRefresh(code)

	// SSE broadcast handles the UI update, so just return 204
	w.WriteHeader(http.StatusNoContent)
}

// AddBot handles adding a bot to the lobby
func (h *LobbyHandler) AddBot(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	_, err := h.botService.AddBotToLobby(r.Context(), code, player.ID)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not add bot: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast member list update
	lob, _ := h.lobbyController.GetLobby(r.Context(), code)
	if lob != nil {
		h.broadcaster.BroadcastMemberListUpdate(r.Context(), lob)
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveBot handles removing a bot from the lobby
func (h *LobbyHandler) RemoveBot(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	botPlayerID := model.PlayerID(r.FormValue("bot_player_id"))
	err := h.botService.RemoveBotFromLobby(r.Context(), code, player.ID, botPlayerID)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not remove bot: "+err.Error())
		w.Header().Set("HX-Redirect", "/lobby/"+string(code))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Broadcast member list update
	lob, _ := h.lobbyController.GetLobby(r.Context(), code)
	if lob != nil {
		h.broadcaster.BroadcastMemberListUpdate(r.Context(), lob)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Events handles SSE event stream for a lobby
func (h *LobbyHandler) Events(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	code := model.LobbyCode(vars["code"])

	// Verify player is in the lobby
	lob, err := h.lobbyController.GetLobby(r.Context(), code)
	if err != nil {
		http.Error(w, "Lobby not found", http.StatusNotFound)
		return
	}

	if lob.GetMember(player.ID) == nil {
		http.Error(w, "Not a member of this lobby", http.StatusForbidden)
		return
	}

	// Get or create hub for this lobby
	hub := h.hubManager.GetOrCreateHub(code)

	// Serve SSE connection
	sse.ServeSSE(w, r, hub, player.ID)
}
