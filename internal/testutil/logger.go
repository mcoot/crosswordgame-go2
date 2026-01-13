package testutil

import (
	"io"
	"log/slog"
)

// NopLogger returns a logger that discards all output.
// Use this in tests to avoid log noise.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
