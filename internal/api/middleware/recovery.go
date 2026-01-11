package middleware

import (
	"log/slog"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/api/apierr"
	"github.com/mcoot/crosswordgame-go2/internal/middleware"
)

// Recovery creates panic recovery middleware for the API
// Returns JSON error responses on panic
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return middleware.Recovery(logger, apiPanicHandler)
}

func apiPanicHandler(w http.ResponseWriter, _ *http.Request, _ any) {
	apierr.WriteError(w, apierr.NewInternalError())
}
