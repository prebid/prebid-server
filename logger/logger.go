package logger

import "context"

// logger is the implementation every function below delegates to. It is built
// with a call depth of 1 rather than through NewGlogLogger, because these
// wrappers put one extra frame between the caller and the logger, and glog would
// otherwise attribute every record in prebid-server to this file.
var logger Logger = newGlogLogger(1)

// FormattedLogger (printf-style) package-level functions.

// Debugf level logging
func Debugf(msg string, args ...any) {
	logger.Debugf(msg, args...)
}

// Infof level logging
func Infof(msg string, args ...any) {
	logger.Infof(msg, args...)
}

// Warnf level logging
func Warnf(msg string, args ...any) {
	logger.Warnf(msg, args...)
}

// Errorf level logging
func Errorf(msg string, args ...any) {
	logger.Errorf(msg, args...)
}

// Fatalf level logging and terminates the program execution.
func Fatalf(msg string, args ...any) {
	logger.Fatalf(msg, args...)
}

// StructuredLogger (slog-style) package-level functions. These mirror the
// printf-style functions above so callers can reach the structured API without
// holding a Logger value, and take key/value pairs rather than a format string.

// Debug logs at Debug level with structured key/value arguments.
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// DebugContext logs at Debug level with context and structured key/value arguments.
func DebugContext(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

// Info logs at Info level with structured key/value arguments.
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// InfoContext logs at Info level with context and structured key/value arguments.
func InfoContext(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

// Warn logs at Warn level with structured key/value arguments.
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// WarnContext logs at Warn level with context and structured key/value arguments.
func WarnContext(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}

// Error logs at Error level with structured key/value arguments.
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// ErrorContext logs at Error level with context and structured key/value arguments.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}

// Exiter package-level functions. See the Exiter interface for the termination
// contract, including the exit hooks these run before the process dies.

// Fatal logs at Fatal level with structured key/value arguments, then terminates
// the program execution.
func Fatal(msg string, args ...any) {
	logger.Fatal(msg, args...)
}

// FatalContext logs at Fatal level with context and structured key/value
// arguments, then terminates the program execution.
func FatalContext(ctx context.Context, msg string, args ...any) {
	logger.FatalContext(ctx, msg, args...)
}
