package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Level represents the logging level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	// Default is the default logger instance
	Default *slog.Logger
)

func init() {
	// Initialize with default configuration
	Default = New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stderr,
	})
}

// Format represents the output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds logger configuration
type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

// New creates a new logger with the given configuration
func New(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: toSlogLevel(cfg.Level),
		// Add source location to logs
		AddSource: false,
	}

	var handler slog.Handler
	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// toSlogLevel converts our Level to slog.Level
func toSlogLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = Default
	}
	return logger.With()
}

// Debug logs a debug message using the default logger
func Debug(msg string, args ...any) {
	Default.Debug(msg, args...)
}

// Info logs an info message using the default logger
func Info(msg string, args ...any) {
	Default.Info(msg, args...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, args ...any) {
	Default.Warn(msg, args...)
}

// Error logs an error message using the default logger
func Error(msg string, args ...any) {
	Default.Error(msg, args...)
}

// With creates a child logger with the given attributes
func With(args ...any) *slog.Logger {
	return Default.With(args...)
}
