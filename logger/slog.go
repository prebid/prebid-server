package logger

import (
	"context"
	"log/slog"
)

// SlogLogger implements the Logger interface for logging using the slog library.
type SlogLogger struct {
	logger *slog.Logger
}

// Debug logs a debug-level message with the specified format and arguments.
func (s *SlogLogger) Debug(msg any, args ...any) {
	s.logger.Debug(convertToString(msg), args...)
}

// DebugContext logs a debug-level message with context, using the specified format and optional arguments.
func (s *SlogLogger) DebugContext(ctx context.Context, msg any, args ...any) {
	s.logger.DebugContext(ctxOrBg(ctx), convertToString(msg), args...)
}

// Info logs an informational-level message with the specified format and optional arguments.
func (s *SlogLogger) Info(msg any, args ...any) {
	s.logger.Info(convertToString(msg), args...)
}

// InfoContext logs an informational-level message with context, using the specified message and optional arguments.
func (s *SlogLogger) InfoContext(ctx context.Context, msg any, args ...any) {
	s.logger.InfoContext(ctxOrBg(ctx), convertToString(msg), args...)
}

// Warn logs a warning-level message with the specified format and arguments.
func (s *SlogLogger) Warn(msg any, args ...any) {
	s.logger.Warn(convertToString(msg), args...)
}

// WarnContext logs a warning-level message with context, using the specified format and optional arguments.
func (s *SlogLogger) WarnContext(ctx context.Context, msg any, args ...any) {
	s.logger.WarnContext(ctxOrBg(ctx), convertToString(msg), args...)
}

// Error logs an error-level message with the specified format and arguments.
func (s *SlogLogger) Error(msg any, args ...any) {
	s.logger.Error(convertToString(msg), args...)
}

// ErrorContext logs an error-level message with context, using the specified format and optional arguments.
func (s *SlogLogger) ErrorContext(ctx context.Context, msg any, args ...any) {
	s.logger.ErrorContext(ctxOrBg(ctx), convertToString(msg), args...)
}

func NewSlogLogger() Logger {
	logger := slog.Default()

	return &SlogLogger{
		logger: logger,
	}
}
