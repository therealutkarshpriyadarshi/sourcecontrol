package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/utkarsh5026/SourceControl/pkg/common/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config logger.Config
	}{
		{
			name: "debug_text",
			config: logger.Config{
				Level:  logger.LevelDebug,
				Format: logger.FormatText,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "info_json",
			config: logger.Config{
				Level:  logger.LevelInfo,
				Format: logger.FormatJSON,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "warn_text",
			config: logger.Config{
				Level:  logger.LevelWarn,
				Format: logger.FormatText,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "error_json",
			config: logger.Config{
				Level:  logger.LevelError,
				Format: logger.FormatJSON,
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(tt.config)
			if log == nil {
				t.Fatal("expected non-nil logger")
			}

			// Test that logger can be used
			log.Info("test message", "key", "value")
		})
	}
}

func TestLogLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	// Debug should not appear (level is Info)
	log.Debug("debug message")
	if strings.Contains(buf.String(), "debug message") {
		t.Error("debug message should not appear at Info level")
	}

	// Info should appear
	buf.Reset()
	log.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("info message should appear at Info level")
	}

	// Warn should appear
	buf.Reset()
	log.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("warn message should appear at Info level")
	}

	// Error should appear
	buf.Reset()
	log.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("error message should appear at Info level")
	}
}

func TestStructuredLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	log.Info("test message", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("message should appear in output")
	}
	if !strings.Contains(output, "key1=value1") {
		t.Error("key1=value1 should appear in output")
	}
	if !strings.Contains(output, "key2=42") {
		t.Error("key2=42 should appear in output")
	}
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatJSON,
		Output: buf,
	})

	log.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Error("message should appear in JSON output")
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Error("key-value pair should appear in JSON output")
	}
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	// Create a child logger with additional context
	componentLogger := baseLogger.With("component", "test", "version", "1.0")

	componentLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Error("component=test should appear in output")
	}
	if !strings.Contains(output, "version=1.0") {
		t.Error("version=1.0 should appear in output")
	}
}

func TestDefaultLogger(t *testing.T) {
	if logger.Default == nil {
		t.Fatal("default logger should not be nil")
	}

	// Test that global functions work
	buf := &bytes.Buffer{}
	logger.Default = logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	logger.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("global Info function should work")
	}
}
