package logger

var logger Logger = NewGlogLogger()

// Debug level logging
func Debugf(msg string, args ...any) {
	logger.Debugf(msg, args...)
}

// Info level logging
func Infof(msg string, args ...any) {
	logger.Infof(msg, args...)
}

// Warn level logging
func Warnf(msg string, args ...any) {
	logger.Warnf(msg, args...)
}

// Error level logging
func Errorf(msg string, args ...any) {
	logger.Errorf(msg, args...)
}

// Fatal level logging and terminates the program execution.
func Fatalf(msg string, args ...any) {
	logger.Fatalf(msg, args...)
}
