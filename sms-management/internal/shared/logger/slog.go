package logger

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Env    string
	Level  string
	Format string
}

func New(cfg Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	isJSON := strings.ToLower(cfg.Format) == "json" ||
		strings.ToLower(cfg.Env) == "production" ||
		strings.ToLower(cfg.Env) == "staging"

	if !isJSON {
		opts.AddSource = true
		handler := slog.NewTextHandler(os.Stdout, opts)
		return slog.New(handler)
	}

	opts.AddSource = false
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
