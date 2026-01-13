package factory

import (
	"io"
	"log/slog"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/clock"
	"github.com/mcoot/crosswordgame-go2/internal/dependencies/random"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/services/board"
	"github.com/mcoot/crosswordgame-go2/internal/services/dictionary"
	"github.com/mcoot/crosswordgame-go2/internal/services/game"
	"github.com/mcoot/crosswordgame-go2/internal/services/lobby"
	"github.com/mcoot/crosswordgame-go2/internal/services/scoring"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// App contains all wired application components
type App struct {
	// Storage
	Storage storage.Storage

	// External dependencies
	Clock  clock.Clock
	Random random.Random

	// Services
	DictionaryService *dictionary.Service
	BoardService      *board.Service
	ScoringService    *scoring.Service
	GameController    *game.Controller
	LobbyController   *lobby.Controller
	AuthService       *auth.Service
	HubManager        *sse.HubManager
}

// Config holds configuration for the application factory
type Config struct {
	// DictionaryPath is the path to the dictionary file (optional)
	// If empty, dictionary must be loaded manually
	DictionaryPath string
	// AuthConfig holds configuration for the auth service (optional)
	// If zero value, defaults to auth.DefaultConfig()
	AuthConfig auth.Config
	// Logger is the application logger (optional)
	// If nil, a no-op logger is used
	Logger *slog.Logger
}

// New creates a new application with all dependencies wired
func New(cfg Config) *App {
	// Create storage
	store := memory.New()

	// Create external dependencies
	clk := clock.New()
	rnd := random.New()

	// Use default auth config if not provided
	authCfg := cfg.AuthConfig
	if authCfg.SessionDuration == 0 {
		authCfg = auth.DefaultConfig()
	}

	// Use no-op logger if not provided
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return newWithDependencies(store, clk, rnd, authCfg, logger)
}

// newWithDependencies creates an App with the given dependencies (useful for testing)
func newWithDependencies(store storage.Storage, clk clock.Clock, rnd random.Random, authCfg auth.Config, logger *slog.Logger) *App {
	// Create services
	dictService := dictionary.New(store, logger)
	boardService := board.New(store, logger)
	scoringService := scoring.New(dictService)
	gameController := game.NewController(store, boardService, scoringService, clk, rnd, logger)
	lobbyController := lobby.NewController(store, gameController, clk, rnd, logger)
	authService := auth.New(store, clk, authCfg, logger)
	hubManager := sse.NewHubManager()

	return &App{
		Storage:           store,
		Clock:             clk,
		Random:            rnd,
		DictionaryService: dictService,
		BoardService:      boardService,
		ScoringService:    scoringService,
		GameController:    gameController,
		LobbyController:   lobbyController,
		AuthService:       authService,
		HubManager:        hubManager,
	}
}
