package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/golang/glog"
	slogglog "github.com/searKing/golang/go/log/slog"
)

// GlogLogger implements the Logger interface for logging using the glog library with configurable call depth.
// It also provides slog-compatible methods through an embedded slog.Logger that uses a glog handler.
type GlogLogger struct {
	depth      int
	slogLogger *slog.Logger
	exitFunc   func(int) // allows testing Fatal without actually exiting
	fatalOut   io.Writer // destination for the Fatal goroutine dump; also the slog handler's sink
}

// Debug logs a debug-level message with the specified format and arguments.
func (logger *GlogLogger) Debugf(msg string, args ...any) {
	glog.InfoDepthf(logger.depth, msg, args...)
}

// Info logs an informational-level message with the specified format and optional arguments.
func (logger *GlogLogger) Infof(msg string, args ...any) {
	glog.InfoDepthf(logger.depth, msg, args...)
}

// Warn logs a warning-level message with the specified format and arguments.
func (logger *GlogLogger) Warnf(msg string, args ...any) {
	glog.WarningDepthf(logger.depth, msg, args...)
}

// Error logs an error-level message with the specified format and arguments.
func (logger *GlogLogger) Errorf(msg string, args ...any) {
	glog.ErrorDepthf(logger.depth, msg, args...)
}

// Fatal logs a fatal-level message with the specified format and arguments, then exits the application.
func (logger *GlogLogger) Fatalf(msg string, args ...any) {
	glog.FatalDepthf(logger.depth, msg, args...)
}

// NewGlogLogger returns a Logger backed by glog. Printf-style methods route
// through glog directly; structured (slog-style) methods are written in glog's
// line format to stderr via a glog handler.
func NewGlogLogger() Logger {
	return newGlogLogger(os.Stderr)
}

// newGlogLogger builds a *GlogLogger whose structured records and fatal goroutine
// dump are written to out. It is split from NewGlogLogger so tests can capture
// that output and exercise the real handler and level-prefix mapping rather than
// a duplicated copy of it.
//
// AddSource is left off deliberately. The Debug/Info/... wrappers add a call hop
// that slog's fixed source-skip depth does not account for, so enabling it would
// record this file's location rather than the caller's. As a result structured
// records omit file:line; the printf-style methods, which route through glog,
// still carry glog's own file:line. Wiring correct source capture is deferred to
// a later phase (it needs the wrapper hop removed).
func newGlogLogger(out io.Writer) *GlogLogger {
	handler := slogglog.NewGlogHandler(out, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Allow all levels, glog will handle filtering
	})

	// Capture the default level-to-prefix mapping, then customize to add "F" for
	// Fatal level, delegating to the default mapping for D/I/W/E.
	defaultReplaceLevelString := handler.ReplaceLevelString
	handler.ReplaceLevelString = func(l slog.Level) string {
		if l >= LevelFatal {
			return "F"
		}
		return defaultReplaceLevelString(l)
	}

	return &GlogLogger{
		depth:      1,
		slogLogger: slog.New(handler),
		exitFunc:   os.Exit, // default to os.Exit, can be overridden for testing
		fatalOut:   out,
	}
}

// StructuredLogger interface implementation

// Debug logs at Debug level using slog
func (logger *GlogLogger) Debug(msg string, args ...any) {
	logger.DebugContext(context.Background(), msg, args...)
}

// DebugContext logs at Debug level with context using slog
func (logger *GlogLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.DebugContext(ctx, msg, args...)
}

// Info logs at Info level using slog
func (logger *GlogLogger) Info(msg string, args ...any) {
	logger.InfoContext(context.Background(), msg, args...)
}

// InfoContext logs at Info level with context using slog
func (logger *GlogLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.InfoContext(ctx, msg, args...)
}

// Warn logs at Warn level using slog
func (logger *GlogLogger) Warn(msg string, args ...any) {
	logger.WarnContext(context.Background(), msg, args...)
}

// WarnContext logs at Warn level with context using slog
func (logger *GlogLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.WarnContext(ctx, msg, args...)
}

// Error logs at Error level using slog
func (logger *GlogLogger) Error(msg string, args ...any) {
	logger.ErrorContext(context.Background(), msg, args...)
}

// ErrorContext logs at Error level with context using slog
func (logger *GlogLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.ErrorContext(ctx, msg, args...)
}

// Fatal logs at Fatal level using slog and terminates the program.
// See FatalContext for the full termination contract.
func (logger *GlogLogger) Fatal(msg string, args ...any) {
	logger.FatalContext(context.Background(), msg, args...)
}

// FatalContext logs at Fatal level with context using slog, then terminates the
// program, mirroring Fatalf's contract (glog.FatalDepthf). Before exiting it:
//  1. writes the structured fatal record (which reaches stderr immediately);
//  2. calls glog.Flush() to flush glog's buffered file sinks, so output from the
//     printf-style methods (which do route through glog) is not lost;
//  3. dumps the stacks of all running goroutines for post-mortem debugging;
//  4. exits with code 2, matching glog's fatal exit code.
//
// The exit is routed through exitFunc so tests can intercept it; the stack dump
// is written to fatalOut (stderr in production). os.Exit runs no deferred
// functions, so this flush-then-dump-then-exit sequence must be explicit.
func (logger *GlogLogger) FatalContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.Log(ctx, LevelFatal, msg, args...)
	glog.Flush()
	dumpAllGoroutines(logger.fatalOut)
	logger.exitFunc(2)
}

// dumpAllGoroutines writes the stack traces of all running goroutines to w,
// mirroring the post-mortem dump glog produces on a fatal error. The buffer is
// grown until the full dump fits.
func dumpAllGoroutines(w io.Writer) {
	if w == nil {
		return
	}
	buf := make([]byte, 64<<10)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	_, _ = w.Write(buf)
}
