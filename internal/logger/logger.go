// Package logger provides structured application logging.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a structured logger with the specified logging level.
func New(level string) *slog.Logger {
	var lvl slog.Level

	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: lvl,
		},
	)

	return slog.New(handler)
}
