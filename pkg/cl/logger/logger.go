package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// LogLevel represents the severity level for logging.
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	ErrorLevel
)

// Logger provides structured logging with level-based filtering.
type Logger interface {
	Debug(v ...any)
	Debugf(format string, a ...any)
	Info(v ...any)
	Infof(format string, a ...any)
	Error(v ...any)
	Errorf(format string, a ...any)
	With(args ...any) Logger
}

type slogLogger struct {
	logger   *slog.Logger
	logLevel LogLevel
}

// New creates a logger with the specified level.
// Alias for NewLogger for convenience.
func New(logLevelStr string) Logger {
	return NewLogger(logLevelStr)
}

// NewLogger creates a logger with the specified level.
// Accepts: "debug", "dbg", "info", "inf", "error", "err" (case-insensitive).
// Defaults to InfoLevel if level string is unrecognized.
// Output format is JSON if LOG_FORMAT=json, otherwise human-readable text.
func NewLogger(logLevelStr string) Logger {
	level := parseLevel(logLevelStr)

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: toSlogLevel(level),
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: toSlogLevel(level),
		})
	}

	return &slogLogger{
		logger:   slog.New(handler),
		logLevel: level,
	}
}

func (l *slogLogger) Debug(v ...any) {
	if l.logLevel <= DebugLevel {
		l.logger.Debug(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Debugf(format string, a ...any) {
	if l.logLevel <= DebugLevel {
		l.logger.Debug(fmt.Sprintf(format, a...))
	}
}

func (l *slogLogger) Info(v ...any) {
	if l.logLevel <= InfoLevel {
		l.logger.Info(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Infof(format string, a ...any) {
	if l.logLevel <= InfoLevel {
		l.logger.Info(fmt.Sprintf(format, a...))
	}
}

func (l *slogLogger) Error(v ...any) {
	if l.logLevel <= ErrorLevel {
		l.logger.Error(fmt.Sprint(v...))
	}
}

func (l *slogLogger) Errorf(format string, a ...any) {
	if l.logLevel <= ErrorLevel {
		l.logger.Error(fmt.Sprintf(format, a...))
	}
}

// With returns a new logger with additional contextual fields.
// The returned logger preserves the current log level.
func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger:   l.logger.With(args...),
		logLevel: l.logLevel,
	}
}

type noopLogger struct{}

func (noopLogger) Debug(v ...any)                 {}
func (noopLogger) Debugf(format string, a ...any) {}
func (noopLogger) Info(v ...any)                  {}
func (noopLogger) Infof(format string, a ...any)  {}
func (noopLogger) Error(v ...any)                 {}
func (noopLogger) Errorf(format string, a ...any) {}
func (noopLogger) With(args ...any) Logger        { return noopLogger{} }

// NewNoopLogger creates a no-op logger that discards all log output.
// Useful for testing or components that don't require logging.
func NewNoopLogger() Logger {
	return noopLogger{}
}

func parseLevel(level string) LogLevel {
	level = strings.ToLower(level)
	switch level {
	case "debug", "dbg":
		return DebugLevel
	case "info", "inf":
		return InfoLevel
	case "error", "err":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
