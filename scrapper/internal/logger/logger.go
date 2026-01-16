package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Level represents the log level.
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Format represents the log output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds logger configuration.
type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

// DefaultConfig returns sensible defaults for logging.
func DefaultConfig() *Config {
	return &Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stdout,
	}
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	config *Config
}

// New creates a new Logger instance.
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse level
	var level slog.Level
	switch strings.ToUpper(string(cfg.Level)) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			}
			return a
		},
	}

	var handler slog.Handler
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	switch strings.ToLower(string(cfg.Format)) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		config: cfg,
	}
}

// SetDefault sets this logger as the default slog logger.
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

// WithContext returns a logger with context attributes.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract any request-scoped values from context if needed
	return l
}

// WithRunID returns a logger with run_id attribute for tracking scrape runs.
func (l *Logger) WithRunID(runID int64) *Logger {
	return &Logger{
		Logger: l.Logger.With("run_id", runID),
		config: l.config,
	}
}

// WithJobID returns a logger with job_id attribute.
func (l *Logger) WithJobID(jobID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("job_id", jobID),
		config: l.config,
	}
}

// WithPage returns a logger with page attribute.
func (l *Logger) WithPage(page int) *Logger {
	return &Logger{
		Logger: l.Logger.With("page", page),
		config: l.config,
	}
}

// WithComponent returns a logger with component attribute.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
	}
}

// ScrapeStarted logs the start of a scrape operation.
func (l *Logger) ScrapeStarted(runID int64, strategy string, maxPages int, cantons []string) {
	l.Info("scrape started",
		"run_id", runID,
		"strategy", strategy,
		"max_pages", maxPages,
		"cantons", cantons,
	)
}

// ScrapeCompleted logs the completion of a scrape operation.
func (l *Logger) ScrapeCompleted(runID int64, status string, processed, inserted, updated, skipped int) {
	l.Info("scrape completed",
		"run_id", runID,
		"status", status,
		"jobs_processed", processed,
		"jobs_inserted", inserted,
		"jobs_updated", updated,
		"jobs_skipped", skipped,
	)
}

// PageFetched logs a successfully fetched page.
func (l *Logger) PageFetched(runID int64, page int, jobCount int) {
	l.Info("page fetched",
		"run_id", runID,
		"page", page,
		"job_count", jobCount,
	)
}

// JobProcessed logs a processed job.
func (l *Logger) JobProcessed(runID int64, jobID string, action string) {
	l.Debug("job processed",
		"run_id", runID,
		"job_id", jobID,
		"action", action,
	)
}

// MigrationStarted logs the start of a migration.
func (l *Logger) MigrationStarted(direction string) {
	l.Info("migration started", "direction", direction)
}

// MigrationCompleted logs the completion of a migration.
func (l *Logger) MigrationCompleted(direction string, version uint) {
	l.Info("migration completed", "direction", direction, "version", version)
}
