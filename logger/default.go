package logger

import (
	"context"

	"github.com/golang/glog"
)

type GlogLogger struct {
	depth int
}

var _ Logger = (*GlogLogger)(nil)

func (logger *GlogLogger) Info(args ...any) {
	glog.InfoDepth(logger.depth, args...)
}

func (logger *GlogLogger) Infof(format string, args ...any) {
	glog.InfoDepthf(logger.depth, format, args...)
}

func (logger *GlogLogger) Warning(args ...any) {
	glog.WarningDepth(logger.depth, args...)
}

func (logger *GlogLogger) Warningf(format string, args ...any) {
	glog.WarningDepthf(logger.depth, format, args...)
}

func (logger *GlogLogger) Error(args ...any) {
	glog.ErrorDepth(logger.depth, args...)
}

func (logger *GlogLogger) Errorf(format string, args ...any) {
	glog.ErrorDepthf(logger.depth, format, args...)
}

func (logger *GlogLogger) Exitf(format string, args ...any) {
	glog.ExitDepthf(logger.depth, format, args...)
}

func (logger *GlogLogger) Fatal(args ...any) {
	glog.FatalDepth(logger.depth, args...)
}

func (logger *GlogLogger) Fatalf(format string, args ...any) {
	glog.FatalDepthf(logger.depth, format, args...)
}

func (logger *GlogLogger) Debugs(msg string, args ...any) {
	// glog doesn't have Debug; just call info
	logger.Infos(msg, args...)
}

func (logger *GlogLogger) DebugsContext(ctx context.Context, msg string, args ...any) {
	// glog doesn't have Debug; just call info
	logger.InfosContext(ctx, msg, args...)
}

func (logger *GlogLogger) Infos(msg string, args ...any) {
	glog.Info(append([]any{msg}, args...)...)
}

func (logger *GlogLogger) InfosContext(ctx context.Context, msg string, args ...any) {
	glog.InfoContext(ctx, append([]any{msg}, args...)...)
}

func (logger *GlogLogger) Warns(msg string, args ...any) {
	glog.Warning(append([]any{msg}, args...)...)
}

func (logger *GlogLogger) WarnsContext(ctx context.Context, msg string, args ...any) {
	glog.WarningContext(ctx, append([]any{msg}, args...)...)
}

func (logger *GlogLogger) Errors(msg string, args ...any) {
	glog.Error(append([]any{msg}, args...)...)
}

func (logger *GlogLogger) ErrorsContext(ctx context.Context, msg string, args ...any) {
	glog.ErrorContext(ctx, append([]any{msg}, args...)...)
}

func NewDefaultLogger(depth int) Logger {
	return &GlogLogger{
		depth: depth,
	}
}
