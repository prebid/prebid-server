package logger

type Logger interface {
	Info(args ...any)
	Infof(format string, args ...any)

	Warning(args ...any)
	Warningf(format string, args ...any)

	Error(args ...any)
	Errorf(format string, args ...any)

	Exitf(format string, args ...any)

	Fatal(args ...any)
	Fatalf(format string, args ...any)
}
