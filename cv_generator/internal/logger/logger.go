package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Level represents logging level.
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Format represents logging format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds logger configuration.
type Config struct {
	Level  Level
	Format Format
}

// Logger wraps slog.Logger.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger.
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

// SetDefault sets this logger as the default slog logger.
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}
