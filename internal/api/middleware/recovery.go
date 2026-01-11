package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/mcoot/crosswordgame-go2/internal/api/apierr"
)

// Recovery creates panic recovery middleware
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						slog.Any("error", err),
						slog.String("stack", string(debug.Stack())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					apierr.WriteError(w, apierr.NewInternalError())
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
