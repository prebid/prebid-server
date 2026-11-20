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
// Termination contract: implementations must (1) run the registered exit hooks
// via RunExitHooks, (2) flush their own buffered sinks so the fatal record and
// prior output are not lost, and (3) dump the stacks of all running goroutines
// to aid post-mortem debugging, before exiting — matching the behavior of
// FormattedLogger.Fatalf. GlogLogger gets (2) and (3) from glog itself.
//
// Termination is performed via os.Exit, which bypasses deferred functions,
// os/signal handlers, and runtime finalizers; Go has no shutdown-hook registry
// of its own (no equivalent of Java's Runtime.addShutdownHook). The exit-hook
// registry in this package fills that gap: cleanup that must survive a fatal
// error — flushing a telemetry exporter, closing an analytics module — should be
// registered with RegisterExitHook rather than deferred, because a defer will
// simply not run.
//
// The hooks are best-effort and bounded by a cleanup budget, so Fatal remains
// what it was: the response to an unrecoverable error, typically at startup. For
// the normal lifecycle, keep driving shutdown from signal handling and explicit
// teardown; RunExitHooks can be called from there too, and the hooks run at most
// once per process either way.
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
