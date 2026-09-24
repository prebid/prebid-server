package logger

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang/glog"
)

// How many frames each path puts between the caller and the call into glog.
// GlogLogger.depth is added on top of these; see its documentation.
//
// Both are pinned by tests that assert on the file:line glog actually reports
// (TestGlogLogger_FormattedOutputAttribution and
// TestGlogLogger_StructuredOutputGoesThroughGlog), so neither can silently drift.
const (
	// formattedCallDepth covers the printf-style method itself, which calls
	// glog.*Depthf directly.
	formattedCallDepth = 1

	// structuredCallDepth covers the structured method, GlogLogger.log,
	// glogHandler.Handle and emitToGlog.
	structuredCallDepth = 4
)

// GlogLogger implements the Logger interface on top of glog, with configurable
// call depth. Both the printf-style methods and the structured (slog-style) ones
// end up in glog, so a message logged either way lands in the same files, obeys
// the same flags and carries the same file:line.
type GlogLogger struct {
	// depth is how many wrapper frames sit between an exported method of this
	// type and the call site whose file:line should be reported: 0 for a logger
	// called directly, 1 for the one behind the package-level functions in
	// logger.go.
	depth   int
	handler *glogHandler

	// fatalDepthf is a test seam for the printf-style fatal path; nil means
	// glog.FatalDepthf. The structured fatal path is intercepted through the
	// handler's own emit seam instead.
	fatalDepthf func(depth int, format string, args ...any)
}

// NewGlogLogger returns a Logger backed by glog, attributing each record to
// whoever called it directly. The package-level functions in logger.go add a
// frame of their own, so they use their own instance; see logger.go.
func NewGlogLogger() Logger {
	return newGlogLogger(0)
}

// newGlogLogger builds a *GlogLogger for a call site sitting depth wrapper
// frames above its methods.
func newGlogLogger(depth int) *GlogLogger {
	return &GlogLogger{
		depth:   depth,
		handler: &glogHandler{depth: depth + structuredCallDepth},
	}
}

// FormattedLogger interface implementation. These are unchanged passthroughs to
// glog, other than Fatalf running the exit hooks.

// Debugf logs a debug-level message with the specified format and arguments.
// glog has no debug severity, so it is recorded at INFO.
func (logger *GlogLogger) Debugf(msg string, args ...any) {
	glog.InfoDepthf(logger.depth+formattedCallDepth, msg, args...)
}

// Infof logs an informational-level message with the specified format and optional arguments.
func (logger *GlogLogger) Infof(msg string, args ...any) {
	glog.InfoDepthf(logger.depth+formattedCallDepth, msg, args...)
}

// Warnf logs a warning-level message with the specified format and arguments.
func (logger *GlogLogger) Warnf(msg string, args ...any) {
	glog.WarningDepthf(logger.depth+formattedCallDepth, msg, args...)
}

// Errorf logs an error-level message with the specified format and arguments.
func (logger *GlogLogger) Errorf(msg string, args ...any) {
	glog.ErrorDepthf(logger.depth+formattedCallDepth, msg, args...)
}

// Fatalf logs a fatal-level message with the specified format and arguments,
// then terminates the application. See the Exiter interface for the termination
// contract; the registered exit hooks run first, then glog.FatalDepthf writes
// the record to every severity file, dumps the stacks of all goroutines, flushes
// and exits 2.
func (logger *GlogLogger) Fatalf(msg string, args ...any) {
	_ = RunExitHooks(context.Background())
	fatalDepthf := logger.fatalDepthf
	if fatalDepthf == nil {
		fatalDepthf = glog.FatalDepthf
	}
	fatalDepthf(logger.depth+formattedCallDepth, msg, args...)
}

// StructuredLogger interface implementation.
//
// Every method calls log directly rather than delegating to its Context variant:
// the frame glog reports is derived from a fixed call depth, so an extra hop
// would attribute the record to this file instead of the caller.

// Debug logs at Debug level with structured key/value arguments.
func (logger *GlogLogger) Debug(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelDebug, msg, args...)
}

// DebugContext logs at Debug level with context and structured key/value arguments.
func (logger *GlogLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelDebug, msg, args...)
}

// Info logs at Info level with structured key/value arguments.
func (logger *GlogLogger) Info(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelInfo, msg, args...)
}

// InfoContext logs at Info level with context and structured key/value arguments.
func (logger *GlogLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelInfo, msg, args...)
}

// Warn logs at Warn level with structured key/value arguments.
func (logger *GlogLogger) Warn(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelWarn, msg, args...)
}

// WarnContext logs at Warn level with context and structured key/value arguments.
func (logger *GlogLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelWarn, msg, args...)
}

// Error logs at Error level with structured key/value arguments.
func (logger *GlogLogger) Error(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelError, msg, args...)
}

// ErrorContext logs at Error level with context and structured key/value arguments.
func (logger *GlogLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelError, msg, args...)
}

// Exiter interface implementation.

// Fatal logs at Fatal level with structured key/value arguments, then terminates
// the program. See FatalContext for the termination contract.
func (logger *GlogLogger) Fatal(msg string, args ...any) {
	_ = RunExitHooks(context.Background())
	logger.log(context.Background(), LevelFatal, msg, args...)
}

// FatalContext logs at Fatal level with context and structured key/value
// arguments, then terminates the program, with the same contract as Fatalf.
//
// The exit hooks run first, before the record is written, because the record is
// written by glog.FatalDepth, which logs, flushes, dumps every goroutine's stack
// and calls os.Exit in one step — there is no point after it at which cleanup
// could run. Nothing is lost by that ordering: the fatal record goes to glog's
// own sinks, which glog flushes itself, and the hooks exist to drain the sinks
// glog does not know about.
//
// The hooks are handed ctx with cancellation removed (context.WithoutCancel), so
// a fatal raised while serving an already-cancelled request still gets a usable
// cleanup budget while keeping ctx's values visible. Because WithoutCancel also
// drops the deadline, the budget is always the one from SetExitHookTimeout.
func (logger *GlogLogger) FatalContext(ctx context.Context, msg string, args ...any) {
	_ = RunExitHooks(context.WithoutCancel(ctx))
	logger.log(ctx, LevelFatal, msg, args...)
}

// log builds one structured record and hands it to the handler.
//
// It deliberately does not go through *slog.Logger. slog captures the caller's
// program counter at a fixed depth that assumes a single wrapper hop, and there
// are two here, so records would be attributed to this file. Emitting through
// the handler directly keeps the frame arithmetic in one place
// (structuredCallDepth) where a test can pin it.
func (logger *GlogLogger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	handler := logger.slogHandler()
	if !handler.Enabled(ctx, level) {
		return
	}
	// The PC is left zero: glog derives file:line from the call depth, so slog's
	// source capture would be redundant work.
	record := slog.NewRecord(time.Now(), level, msg, 0)
	record.Add(args...)
	_ = handler.Handle(ctx, record)
}

// slogHandler returns the handler to emit through, building one on demand for a
// GlogLogger that was constructed as a struct literal rather than through
// NewGlogLogger, so that the zero value stays usable.
func (logger *GlogLogger) slogHandler() *glogHandler {
	if logger.handler != nil {
		return logger.handler
	}
	return &glogHandler{depth: logger.depth + structuredCallDepth}
}
