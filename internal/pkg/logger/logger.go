package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/phsym/console-slog"
)

// New builds a structured logger. Format "console" yields a colored
// human-readable handler; "json" emits JSON. Unknown formats fall back to text.
func New(level, format string) *slog.Logger {
	lvl := parseLevel(level)

	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	case "text":
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	default: // "console"
		h = console.NewHandler(os.Stdout, &console.HandlerOptions{Level: lvl})
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
