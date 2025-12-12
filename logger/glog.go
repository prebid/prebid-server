package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/golang/glog"
	slogglog "github.com/searKing/golang/go/log/slog"
)

// GlogLogger implements the Logger interface for logging using the glog library with configurable call depth.
// It also provides slog-compatible methods through an embedded slog.Logger that uses a glog handler.
type GlogLogger struct {
	depth      int
	slogLogger *slog.Logger
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

func NewGlogLogger() Logger {
	// Create a glog handler that writes to stderr (matching glog's default behavior)
	handler := slogglog.NewGlogHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Allow all levels, glog will handle filtering
	})

	return &GlogLogger{
		depth:      1,
		slogLogger: slog.New(handler),
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

// Fatal logs at Fatal level using slog and terminates the program
func (logger *GlogLogger) Fatal(msg string, args ...any) {
	logger.FatalContext(context.Background(), msg, args...)
}

// FatalContext logs at Fatal level with context using slog and terminates the program
func (logger *GlogLogger) FatalContext(ctx context.Context, msg string, args ...any) {
	logger.slogLogger.Log(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}
