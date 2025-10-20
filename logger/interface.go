package logger

type Logger interface {
	// Debug level logging
	Debug(msg string, args ...any)

	// Info level logging
	Info(msg string, args ...any)

	// Warn level logging
	Warn(msg string, args ...any)

	// Error level logging
	Error(msg string, args ...any)

	// Fatal level logging
	Fatal(msg string, args ...any)
}
