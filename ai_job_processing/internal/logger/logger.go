package logger

import (
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
		Format: FormatJSON,
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

// WithRequestID returns a logger with request_id attribute.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("request_id", requestID),
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

// WithComponent returns a logger with component attribute.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
	}
}

// APIRequest logs an incoming API request.
func (l *Logger) APIRequest(method, path string, statusCode int, duration time.Duration) {
	l.Info("api request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// GeminiRequest logs a Gemini API call.
func (l *Logger) GeminiRequest(operation string, inputTokens, outputTokens int, duration time.Duration) {
	l.Info("gemini request",
		"operation", operation,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"duration_ms", duration.Milliseconds(),
	)
}

// ProcessingStarted logs the start of job processing.
func (l *Logger) ProcessingStarted(jobID string, languages []string) {
	l.Info("processing started",
		"job_id", jobID,
		"target_languages", languages,
	)
}

// ProcessingCompleted logs the completion of job processing.
func (l *Logger) ProcessingCompleted(jobID string, translationsCount int, duration time.Duration) {
	l.Info("processing completed",
		"job_id", jobID,
		"translations", translationsCount,
		"duration_ms", duration.Milliseconds(),
	)
}
