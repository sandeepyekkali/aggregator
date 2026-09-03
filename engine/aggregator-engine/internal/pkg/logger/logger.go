package logger

import (
	"log/slog"
	"os"
	"strings"
)

var Log *slog.Logger

// InitLogger sets up a structured JSON logger writing to stdout.
func InitLogger(env string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(env) {
	case "production", "prod":
		level = slog.LevelInfo
	default:
		level = slog.LevelDebug // Debug level for local/sandbox development
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // Includes caller file name and line number for troubleshooting
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)

	return Log
}
