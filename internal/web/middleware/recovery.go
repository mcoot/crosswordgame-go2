package middleware

import (
	"log/slog"
	"net/http"

	"github.com/mcoot/crosswordgame-go2/internal/middleware"
)

// Recovery creates panic recovery middleware for the web interface
// Returns an HTML error page on panic
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return middleware.Recovery(logger, webPanicHandler)
}

func webPanicHandler(w http.ResponseWriter, _ *http.Request, _ any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Error</title></head>
<body>
<h1>Internal Server Error</h1>
<p>Something went wrong. Please try again later.</p>
<p><a href="/">Return to home</a></p>
</body>
</html>`))
}
