package api

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/api/handler"
	"github.com/mcoot/crosswordgame-go2/internal/api/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
)

// RouterConfig holds configuration for the API router
type RouterConfig struct {
	Logger          *slog.Logger
	AuthService     *auth.Service
	LobbyController *lobby.Controller
	GameController  *game.Controller
	BoardService    *board.Service
}

// NewRouter creates a new API router with all routes configured
func NewRouter(cfg RouterConfig) http.Handler {
	r := mux.NewRouter()

	// Create handlers
	playerHandler := handler.NewPlayerHandler(cfg.AuthService)
	lobbyHandler := handler.NewLobbyHandler(cfg.LobbyController)
	gameHandler := handler.NewGameHandler(cfg.LobbyController, cfg.GameController, cfg.BoardService)

	// Create middleware
	authMiddleware := middleware.Auth(cfg.AuthService)
	optionalAuthMiddleware := middleware.OptionalAuth(cfg.AuthService)
	loggingMiddleware := middleware.Logging(cfg.Logger)
	recoveryMiddleware := middleware.Recovery(cfg.Logger)

	// API subrouter with common middleware
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(recoveryMiddleware)
	api.Use(loggingMiddleware)

	// Player routes (no auth required for creating players/logging in)
	api.HandleFunc("/players/guest", playerHandler.CreateGuest).Methods(http.MethodPost)
	api.HandleFunc("/players/register", playerHandler.Register).Methods(http.MethodPost)
	api.HandleFunc("/players/login", playerHandler.Login).Methods(http.MethodPost)

	// Protected player routes
	playerProtected := api.PathPrefix("/players").Subrouter()
	playerProtected.Use(authMiddleware)
	playerProtected.HandleFunc("/me", playerHandler.GetMe).Methods(http.MethodGet)

	// Lobby routes (all require auth)
	lobbies := api.PathPrefix("/lobbies").Subrouter()
	lobbies.Use(authMiddleware)
	lobbies.HandleFunc("", lobbyHandler.Create).Methods(http.MethodPost)
	lobbies.HandleFunc("/{code}", lobbyHandler.Get).Methods(http.MethodGet)
	lobbies.HandleFunc("/{code}/join", lobbyHandler.Join).Methods(http.MethodPost)
	lobbies.HandleFunc("/{code}/leave", lobbyHandler.Leave).Methods(http.MethodPost)
	lobbies.HandleFunc("/{code}/config", lobbyHandler.UpdateConfig).Methods(http.MethodPatch)
	lobbies.HandleFunc("/{code}/members/{player_id}/role", lobbyHandler.SetRole).Methods(http.MethodPatch)
	lobbies.HandleFunc("/{code}/transfer-host", lobbyHandler.TransferHost).Methods(http.MethodPost)

	// Game routes (all require auth)
	lobbies.HandleFunc("/{code}/game", gameHandler.Start).Methods(http.MethodPost)
	lobbies.HandleFunc("/{code}/game", gameHandler.Get).Methods(http.MethodGet)
	lobbies.HandleFunc("/{code}/game", gameHandler.Abandon).Methods(http.MethodDelete)
	lobbies.HandleFunc("/{code}/game/announce", gameHandler.Announce).Methods(http.MethodPost)
	lobbies.HandleFunc("/{code}/game/place", gameHandler.Place).Methods(http.MethodPost)

	// Health check endpoint (no auth)
	api.HandleFunc("/health", healthHandler).Methods(http.MethodGet)

	// Allow optional auth for lobby viewing (spectators without accounts)
	_ = optionalAuthMiddleware // Reserved for future use

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
