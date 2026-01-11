package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// PanicHandler is a function that handles panics and writes an error response
type PanicHandler func(w http.ResponseWriter, r *http.Request, err any)

// Recovery creates panic recovery middleware with a custom panic handler
func Recovery(logger *slog.Logger, handler PanicHandler) func(http.Handler) http.Handler {
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

					handler(w, r, err)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultPanicHandler returns a simple 500 Internal Server Error
func DefaultPanicHandler(w http.ResponseWriter, _ *http.Request, _ any) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
