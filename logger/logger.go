package logger

var logger Logger = NewGlogLogger()

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
