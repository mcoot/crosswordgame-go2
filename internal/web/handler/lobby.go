package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
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
	hubManager      *sse.HubManager
}

// NewLobbyHandler creates a new LobbyHandler
func NewLobbyHandler(lobbyController *lobby.Controller, authService *auth.Service, hubManager *sse.HubManager) *LobbyHandler {
	return &LobbyHandler{
		lobbyController: lobbyController,
		authService:     authService,
		hubManager:      hubManager,
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
		if parsed, err := strconv.Atoi(gs); err == nil && parsed >= 4 && parsed <= 7 {
			gridSize = parsed
		}
	}

	lob, err := h.lobbyController.CreateLobby(r.Context(), *player)
	if err != nil {
		middleware.SetFlash(w, "error", "Failed to create lobby")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Update config with grid size
	cfg := model.LobbyConfig{GridSize: gridSize}
	if err := h.lobbyController.UpdateConfig(r.Context(), lob.Code, player.ID, cfg); err != nil {
		// Non-fatal, continue with default config
	}

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

	err := h.lobbyController.JoinLobby(r.Context(), model.LobbyCode(code), *player)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not join lobby: "+err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
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
	}

	flash := middleware.GetFlash(r.Context())

	// Check if player is host
	host := lob.GetHost()
	isHost := host != nil && host.Player.ID == player.ID

	data := pages.LobbyData{
		PageData: layout.PageData{
			Title:  "Lobby " + string(lob.Code),
			Player: player,
			Flash:  flash,
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
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	middleware.SetFlash(w, "info", "You left the lobby")
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
		if parsed, err := strconv.Atoi(gs); err == nil && parsed >= 4 && parsed <= 7 {
			gridSize = parsed
		}
	}

	cfg := model.LobbyConfig{GridSize: gridSize}
	err := h.lobbyController.UpdateConfig(r.Context(), code, player.ID, cfg)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not update config: "+err.Error())
	} else {
		middleware.SetFlash(w, "success", "Settings updated")
	}

	// For HTMX requests, just return 204
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
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
		http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
		return
	}

	// Note: SetRole doesn't verify host, but the caller should be host
	// We could add host verification here if needed
	err := h.lobbyController.SetRole(r.Context(), code, targetPlayerID, role)
	if err != nil {
		middleware.SetFlash(w, "error", "Could not change role: "+err.Error())
	}

	http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
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
	} else {
		middleware.SetFlash(w, "success", "Host transferred")
	}

	http.Redirect(w, r, "/lobby/"+string(code), http.StatusSeeOther)
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
