package logger

import (
	"context"
	"log/slog"
)

// LevelFatal is a custom slog level for fatal errors that terminate the program.
// It is defined as slog.LevelError + 4 to be higher than all standard slog levels.
const LevelFatal = slog.LevelError + 4

// FormattedLogger provides traditional printf-style formatted logging methods.
type FormattedLogger interface {
	// Debugf level logging
	Debugf(msg string, args ...any)

	// Infof level logging
	Infof(msg string, args ...any)

	// Warnf level logging
	Warnf(msg string, args ...any)

	// Errorf level logging
	Errorf(msg string, args ...any)

	// Fatalf level logging and terminates the program execution
	Fatalf(msg string, args ...any)
}

// StructuredLogger provides structured logging methods compatible with log/slog,
// including context-aware variants for propagating request context.
type StructuredLogger interface {
	// Debug logs at Debug level
	Debug(msg string, args ...any)

	// DebugContext logs at Debug level with context
	DebugContext(ctx context.Context, msg string, args ...any)

	// Info logs at Info level
	Info(msg string, args ...any)

	// InfoContext logs at Info level with context
	InfoContext(ctx context.Context, msg string, args ...any)

	// Warn logs at Warn level
	Warn(msg string, args ...any)

	// WarnContext logs at Warn level with context
	WarnContext(ctx context.Context, msg string, args ...any)

	// Error logs at Error level
	Error(msg string, args ...any)

	// ErrorContext logs at Error level with context
	ErrorContext(ctx context.Context, msg string, args ...any)

	// Fatal logs at Fatal level and terminates the program execution
	Fatal(msg string, args ...any)

	// FatalContext logs at Fatal level with context and terminates the program execution
	FatalContext(ctx context.Context, msg string, args ...any)
}

// Logger combines both traditional printf-style and modern structured logging interfaces.
// Implementations must provide both formatted logging (FormattedLogger) and structured
// context-aware logging (StructuredLogger).
type Logger interface {
	FormattedLogger
	StructuredLogger
}
