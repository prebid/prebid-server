package logger

import "context"

type Logger interface {
	// Debug level logging
	Debug(msg any, args ...any)
	DebugContext(ctx context.Context, msg any, args ...any)

	// Info level logging
	Info(msg any, args ...any)
	InfoContext(ctx context.Context, msg any, args ...any)

	// Warn level logging
	Warn(msg any, args ...any)
	WarnContext(ctx context.Context, msg any, args ...any)

	// Error level logging
	Error(msg any, args ...any)
	ErrorContext(ctx context.Context, msg any, args ...any)
}
