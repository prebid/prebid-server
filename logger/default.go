package logger

import (
	"github.com/golang/glog"
)

type GlogLogger struct {
	depth int
}

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

func NewDefaultLogger(depth int) Logger {
	return &GlogLogger{
		depth: depth,
	}
}
