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

	// Fatalf logs at fatal level, then terminates the process. See Exiter for the
	// process-termination contract; Fatalf is the printf-style counterpart of
	// Exiter.Fatal and shares that contract.
	Fatalf(msg string, args ...any)
}

// StructuredLogger provides structured logging methods compatible with log/slog,
// including context-aware variants for propagating request context.
//
// It intentionally mirrors the shape of *slog.Logger and is free of
// process-control methods, so a consumer that only needs to emit structured
// records (a test recorder, a fan-out adapter, a request-scoped sub-logger) can
// depend on this interface without reasoning about process termination. Fatal
// behavior lives in the separate Exiter interface.
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
}

// Exiter provides structured logging methods that terminate the process after
// logging. It is separated from StructuredLogger so that structured-logging
// consumers need not depend on process-control behavior.
//
// Termination contract: implementations must, before exiting, (1) flush any
// buffered log sinks so the fatal record and prior buffered output are not lost,
// and (2) dump the stacks of all running goroutines to aid post-mortem
// debugging — matching the behavior of FormattedLogger.Fatalf.
//
// Because termination is performed via os.Exit, it bypasses deferred functions,
// os/signal handlers, and runtime finalizers — Go has no global shutdown-hook
// registry equivalent to Java's Runtime.addShutdownHook. Fatal is therefore for
// unrecoverable errors (typically at startup) where no graceful cleanup is
// possible. For the normal lifecycle, drive shutdown from signal.NotifyContext
// (SIGINT/SIGTERM) and run cleanup explicitly; do not rely on Fatal to release
// resources.
type Exiter interface {
	// Fatal logs at fatal level, then terminates the program execution.
	Fatal(msg string, args ...any)

	// FatalContext logs at fatal level with context, then terminates the program execution.
	FatalContext(ctx context.Context, msg string, args ...any)
}

// Logger combines traditional printf-style logging (FormattedLogger), modern
// structured context-aware logging (StructuredLogger), and fatal/terminating
// methods (Exiter). Implementations must provide all three.
type Logger interface {
	FormattedLogger
	StructuredLogger
	Exiter
}
