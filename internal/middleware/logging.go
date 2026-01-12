package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// ResponseWriter wraps http.ResponseWriter to capture the status code and size
type ResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write captures the response size
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Status returns the captured status code
func (rw *ResponseWriter) Status() int {
	return rw.status
}

// Size returns the captured response size
func (rw *ResponseWriter) Size() int {
	return rw.size
}

// Flush implements http.Flusher for SSE support
func (rw *ResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Logging creates logging middleware that logs HTTP requests
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &ResponseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.status),
				slog.Int("size", wrapped.size),
				slog.Duration("duration", duration),
			)
		})
	}
}
