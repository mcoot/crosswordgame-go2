package factory

import (
	"errors"
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
	redisstorage "github.com/mcoot/crosswordgame-go2/internal/storage/redis"
	"github.com/mcoot/crosswordgame-go2/internal/web/sse"
)

// Storage type constants
const (
	StorageTypeMemory = "memory"
	StorageTypeRedis  = "redis"
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
	// StorageType selects the storage backend ("memory" or "redis")
	// If empty, defaults to "memory"
	StorageType string
	// RedisConfig holds Redis connection settings (required if StorageType is "redis")
	RedisConfig *redisstorage.Config
}

// New creates a new application with all dependencies wired
func New(cfg Config) (*App, error) {
	// Use no-op logger if not provided
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	// Create storage based on type
	var store storage.Storage
	storageType := cfg.StorageType
	if storageType == "" {
		storageType = StorageTypeMemory
	}

	switch storageType {
	case StorageTypeMemory:
		store = memory.New()
	case StorageTypeRedis:
		if cfg.RedisConfig == nil {
			return nil, errors.New("RedisConfig required when StorageType is redis")
		}
		redisStore, err := redisstorage.New(*cfg.RedisConfig)
		if err != nil {
			return nil, err
		}
		store = redisStore
	default:
		return nil, errors.New("invalid StorageType: must be 'memory' or 'redis'")
	}

	// Create external dependencies
	clk := clock.New()
	rnd := random.New()

	// Use default auth config if not provided
	authCfg := cfg.AuthConfig
	if authCfg.SessionDuration == 0 {
		authCfg = auth.DefaultConfig()
	}

	return newWithDependencies(store, clk, rnd, authCfg, logger), nil
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
	hubManager := sse.NewHubManager(logger)

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
