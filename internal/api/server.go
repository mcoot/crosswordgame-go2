package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ServerConfig holds configuration for the HTTP server
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DefaultServerConfig returns sensible defaults for server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            "",
		Port:            8080,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    60 * time.Second, // Long timeout for SSE (keepalive is 15s)
		ShutdownTimeout: 30 * time.Second,
	}
}

// Server wraps the HTTP server with graceful shutdown support
type Server struct {
	server *http.Server
	logger *slog.Logger
	config ServerConfig
}

// NewServer creates a new API server
func NewServer(handler http.Handler, config ServerConfig, logger *slog.Logger) *Server {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		},
		logger: logger,
		config: config,
	}
}

// Start begins listening for HTTP requests
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", slog.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

// Addr returns the server's listen address
func (s *Server) Addr() string {
	return s.server.Addr
}
