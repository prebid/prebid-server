package logger

import (
	"context"

	"github.com/golang/glog"
)

// GlogLogger implements the Logger interface for logging using the glog library with configurable call depth.
type GlogLogger struct {
	depth int
}

// Debug logs a debug-level message with the specified format and arguments.
func (logger *GlogLogger) Debug(msg any, args ...any) {
	glog.InfoDepthf(logger.depth, convertToString(msg, args...))
}

// DebugContext logs a debug-level message with context, using the specified format and arguments.
func (logger *GlogLogger) DebugContext(ctx context.Context, msg any, args ...any) {
	glog.InfoContextDepthf(ctx, logger.depth, convertToString(msg, args...))
}

// Info logs an informational-level message with the specified format and optional arguments.
func (logger *GlogLogger) Info(msg any, args ...any) {
	glog.InfoDepthf(logger.depth, convertToString(msg, args...))
}

// InfoContext logs an informational-level message with context, using the specified format and optional arguments.
func (logger *GlogLogger) InfoContext(ctx context.Context, msg any, args ...any) {
	glog.InfoContextDepthf(ctx, logger.depth, convertToString(msg, args...))
}

// Warn logs a warning-level message with the specified format and arguments.
func (logger *GlogLogger) Warn(msg any, args ...any) {
	glog.WarningDepthf(logger.depth, convertToString(msg, args...))
}

// WarnContext logs a warning-level message with context, using the specified format and optional arguments.
func (logger *GlogLogger) WarnContext(ctx context.Context, msg any, args ...any) {
	glog.WarningContextDepthf(ctx, logger.depth, convertToString(msg, args...))
}

// Error logs an error-level message with the specified format and arguments.
func (logger *GlogLogger) Error(msg any, args ...any) {
	glog.ErrorDepthf(logger.depth, convertToString(msg, args...))
}

// ErrorContext logs an error-level message with context, using the specified format and optional arguments.
func (logger *GlogLogger) ErrorContext(ctx context.Context, msg any, args ...any) {
	glog.ErrorContextDepthf(ctx, logger.depth, convertToString(msg, args...))
}

func NewGlogLogger(depth int) Logger {
	return &GlogLogger{
		depth: depth,
	}
}
