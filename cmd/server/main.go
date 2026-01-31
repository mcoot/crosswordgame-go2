package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mcoot/crosswordgame-go2/internal/api"
	"github.com/mcoot/crosswordgame-go2/internal/factory"
	redisstorage "github.com/mcoot/crosswordgame-go2/internal/storage/redis"
	"github.com/mcoot/crosswordgame-go2/internal/web"
)

func main() {
	// Set up logging with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Build factory config from environment
	cfg := factory.Config{
		DictionaryPath: "data/words.txt",
		Logger:         logger,
		StorageType:    os.Getenv("STORAGE_TYPE"),
	}

	// Configure Redis if storage type is redis
	if cfg.StorageType == factory.StorageTypeRedis {
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			logger.Error("REDIS_URL required when STORAGE_TYPE=redis")
			os.Exit(1)
		}
		redisCfg := redisstorage.DefaultConfig()
		redisCfg.URL = redisURL
		cfg.RedisConfig = &redisCfg
	}

	// Create application factory
	app, err := factory.New(cfg)
	if err != nil {
		logger.Error("failed to create application", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Load dictionary
	if err := app.DictionaryService.LoadFromFile(context.Background(), "data/words.txt"); err != nil {
		logger.Warn("could not load dictionary", slog.String("error", err.Error()))
	}

	// Find static files directory
	staticDir := findStaticDir()

	// Create API router
	apiRouter := api.NewRouter(api.RouterConfig{
		Logger:          logger,
		AuthService:     app.AuthService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		BotService:      app.BotService,
		HubManager:      app.HubManager,
	})

	// Create web router
	webRouter := web.NewRouter(web.RouterConfig{
		Logger:          logger,
		AuthService:     app.AuthService,
		LobbyController: app.LobbyController,
		GameController:  app.GameController,
		BoardService:    app.BoardService,
		ScoringService:  app.ScoringService,
		BotService:      app.BotService,
		HubManager:      app.HubManager,
		StaticDir:       staticDir,
	})

	// Combine routers
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	// Create server
	serverConfig := api.DefaultServerConfig()
	server := api.NewServer(mux, serverConfig, logger)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutdown signal received")
		cancel()
	}()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	logger.Info("server started", slog.String("addr", server.Addr()))

	// Wait for shutdown or error
	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case <-ctx.Done():
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("shutdown error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	logger.Info("server stopped")
}

// findStaticDir looks for the static files directory
func findStaticDir() string {
	// Try common locations
	candidates := []string{
		"internal/web/static",
		"./internal/web/static",
		filepath.Join(os.Getenv("PWD"), "internal/web/static"),
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	// Default to relative path
	return "internal/web/static"
}
