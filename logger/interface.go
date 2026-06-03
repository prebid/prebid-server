package logger

type Logger interface {
	// Debugf level logging
	Debugf(msg string, args ...any)

	// Infof level logging
	Infof(msg string, args ...any)

	// Warnf level logging
	Warnf(msg string, args ...any)

	// Errorf level logging
	Errorf(msg string, args ...any)

	// Fatalf level logging
	Fatalf(msg string, args ...any)
}
