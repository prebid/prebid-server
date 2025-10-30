package logger

type Logger interface {
	// Debug level logging
	Debugf(msg string, args ...any)

	// Info level logging
	Infof(msg string, args ...any)

	// Warn level logging
	Warnf(msg string, args ...any)

	// Error level logging
	Errorf(msg string, args ...any)

	// Fatal level logging
	Fatalf(msg string, args ...any)
}
