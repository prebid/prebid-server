package logger

import "context"

type (
	PrintfLogger interface {
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
	StructuredLogger interface {
		Debugs(msg string, args ...any)
		DebugsContext(ctx context.Context, msg string, args ...any)

		Infos(msg string, args ...any)
		InfosContext(ctx context.Context, msg string, args ...any)

		Warns(msg string, args ...any)
		WarnsContext(ctx context.Context, msg string, args ...any)

		Errors(msg string, args ...any)
		ErrorsContext(ctx context.Context, msg string, args ...any)
	}
	Logger interface {
		PrintfLogger
		StructuredLogger
	}
)
