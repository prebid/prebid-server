package logger

var defaultDepth int = 1

var logger Logger = ProvideDefaultLogger(defaultDepth)

func New(loggerType string, depth *int) {
	if depth == nil {
		depth = &defaultDepth
	}

	switch loggerType {
	case "alternative":
		logger = ProvideAlternativeLogger(*depth)
	default:
		logger = ProvideDefaultLogger(*depth)
	}
}

func Info(args ...any) {
	logger.Info(args...)
}

func Infof(format string, args ...any) {
	logger.Infof(format, args)
}

func Warning(args ...any) {
	logger.Warning(args...)
}

func Warningf(format string, args ...any) {
	logger.Warningf(format, args)
}

func Warningln(args ...any) {
	logger.Warningln(args...)
}

func Error(args ...any) {
	logger.Error(args...)
}

func Errorf(format string, args ...any) {
	logger.Errorf(format, args)
}

func Exitf(format string, args ...any) {
	logger.Exitf(format, args)
}

func Fatal(args ...any) {
	logger.Fatal(args...)
}

func Fatalf(format string, args ...any) {
	logger.Fatalf(format, args)
}
