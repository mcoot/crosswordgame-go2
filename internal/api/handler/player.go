package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/api/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/api/request"
	"github.com/mcoot/crosswordgame-go2/internal/api/response"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
)

// PlayerHandler handles player-related endpoints
type PlayerHandler struct {
	authService *auth.Service
}

// NewPlayerHandler creates a new player handler
func NewPlayerHandler(authService *auth.Service) *PlayerHandler {
	return &PlayerHandler{
		authService: authService,
	}
}

// CreateGuest handles POST /api/v1/players/guest
func (h *PlayerHandler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	var req request.CreateGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	if req.DisplayName == "" {
		WriteError(w, NewInvalidRequestError("display_name is required"))
		return
	}

	session, err := h.authService.CreateGuestPlayer(r.Context(), req.DisplayName)
	if err != nil {
		WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, response.AuthResponseFromSession(session))
}

// Register handles POST /api/v1/players/register
func (h *PlayerHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	if req.Username == "" {
		WriteError(w, NewInvalidRequestError("username is required"))
		return
	}
	if req.Password == "" {
		WriteError(w, NewInvalidRequestError("password is required"))
		return
	}
	if req.DisplayName == "" {
		WriteError(w, NewInvalidRequestError("display_name is required"))
		return
	}

	session, err := h.authService.RegisterPlayer(r.Context(), req.Username, req.Password, req.DisplayName)
	if err != nil {
		WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, response.AuthResponseFromSession(session))
}

// Login handles POST /api/v1/players/login
func (h *PlayerHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid request body"))
		return
	}

	if req.Username == "" {
		WriteError(w, NewInvalidRequestError("username is required"))
		return
	}
	if req.Password == "" {
		WriteError(w, NewInvalidRequestError("password is required"))
		return
	}

	session, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, response.AuthResponseFromSession(session))
}

// GetMe handles GET /api/v1/players/me
func (h *PlayerHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	player := middleware.MustGetPlayer(r.Context())
	response.JSON(w, http.StatusOK, response.PlayerFromModel(player))
}
