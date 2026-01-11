package middleware

import (
	"log/slog"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/middleware"
)

// Logging creates logging middleware for the web interface
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return middleware.Logging(logger)
}
