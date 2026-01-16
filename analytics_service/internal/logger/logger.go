package logger

import (
	"log/slog"
	"os"
	"strings"
)

type Level string
type Format string

const (
	LevelDebug Level  = "DEBUG"
	LevelInfo  Level  = "INFO"
	FormatJSON Format = "json"
)

type Config struct {
	Level  Level
	Format Format
}

type Logger struct {
	*slog.Logger
}

func New(cfg *Config) *Logger {
	var level slog.Level
	switch strings.ToUpper(string(cfg.Level)) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(string(cfg.Format)) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{Logger: slog.New(handler)}
}

func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}
