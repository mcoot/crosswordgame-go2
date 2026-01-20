package web

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/web/handler"
	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// RouterConfig holds configuration for the web router
type RouterConfig struct {
	Logger          *slog.Logger
	AuthService     *auth.Service
	LobbyController *lobby.Controller
	GameController  *game.Controller
	BoardService    *board.Service
	ScoringService  *scoring.Service
	HubManager      *sse.HubManager
	StaticDir       string // Path to static files directory
}

// NewRouter creates a new web router with all routes configured
func NewRouter(cfg RouterConfig) http.Handler {
	r := mux.NewRouter()

	// Create middleware
	loggingMiddleware := middleware.Logging(cfg.Logger)
	recoveryMiddleware := middleware.Recovery(cfg.Logger)
	flashMiddleware := middleware.Flash()
	authMiddleware := middleware.Auth(cfg.AuthService)
	optionalAuthMiddleware := middleware.OptionalAuth(cfg.AuthService)

	// Apply global middleware to all routes
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)

	// Create SSE hub manager if not provided
	hubManager := cfg.HubManager
	if hubManager == nil {
		hubManager = sse.NewHubManager(cfg.Logger)
	}

	// Create handlers
	homeHandler := handler.NewHomeHandler()
	authHandler := handler.NewAuthHandler(cfg.AuthService)
	lobbyHandler := handler.NewLobbyHandler(cfg.LobbyController, cfg.AuthService, hubManager, cfg.Logger)
	gameHandler := handler.NewGameHandler(cfg.LobbyController, cfg.GameController, cfg.BoardService, cfg.ScoringService, hubManager, cfg.Logger)

	// Static files
	if cfg.StaticDir != "" {
		staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir)))
		r.PathPrefix("/static/").Handler(staticHandler)
	}

	// Public routes (optional auth for showing player info in nav)
	public := r.NewRoute().Subrouter()
	public.Use(flashMiddleware)
	public.Use(optionalAuthMiddleware)
	public.HandleFunc("/", homeHandler.Home).Methods(http.MethodGet)

	// Auth actions (no auth required)
	authRoutes := r.PathPrefix("/auth").Subrouter()
	authRoutes.Use(flashMiddleware)
	authRoutes.Use(optionalAuthMiddleware)
	authRoutes.HandleFunc("/guest", authHandler.CreateGuest).Methods(http.MethodPost)
	authRoutes.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)

	// Protected routes (require auth)
	protected := r.NewRoute().Subrouter()
	protected.Use(flashMiddleware)
	protected.Use(authMiddleware)

	// Lobby routes
	protected.HandleFunc("/lobby", lobbyHandler.Create).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/join", lobbyHandler.JoinByForm).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}", lobbyHandler.View).Methods(http.MethodGet)
	protected.HandleFunc("/lobby/{code}/leave", lobbyHandler.Leave).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/config", lobbyHandler.UpdateConfig).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/role", lobbyHandler.SetRole).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/transfer-host", lobbyHandler.TransferHost).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/events", lobbyHandler.Events).Methods(http.MethodGet)

	// Game routes
	protected.HandleFunc("/lobby/{code}/game", gameHandler.View).Methods(http.MethodGet)
	protected.HandleFunc("/lobby/{code}/game/start", gameHandler.Start).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/game/announce", gameHandler.Announce).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/game/place", gameHandler.Place).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/game/abandon", gameHandler.Abandon).Methods(http.MethodPost)
	protected.HandleFunc("/lobby/{code}/game/dismiss", gameHandler.Dismiss).Methods(http.MethodPost)

	return r
}
