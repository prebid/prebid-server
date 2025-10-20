package logger

var logger Logger = NewGlogLogger()

// Debug level logging
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info level logging
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn level logging
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error level logging
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// Fatal level logging and terminates the program execution.
func Fatal(msg string, args ...any) {
	logger.Fatal(msg, args...)
}
