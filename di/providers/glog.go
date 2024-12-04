//go:build !custom_logger

package providers

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/di/interfaces"
)

type GlogWrapper struct {
	depth int
}

func (logger *GlogWrapper) Info(args ...any) {
	glog.InfoDepth(logger.depth, args...)
}

func (logger *GlogWrapper) Infof(format string, args ...any) {
	glog.InfoDepthf(logger.depth, format, args...)
}

func (logger *GlogWrapper) Warning(args ...any) {
	glog.WarningDepth(logger.depth, args...)
}

func (logger *GlogWrapper) Warningf(format string, args ...any) {
	glog.WarningDepthf(logger.depth, format, args...)
}

func (logger *GlogWrapper) Warningln(args ...any) {
	glog.WarningDepth(logger.depth, args...)
}

func (logger *GlogWrapper) Error(args ...any) {
	glog.ErrorDepth(logger.depth, args...)
}

func (logger *GlogWrapper) Errorf(format string, args ...any) {
	glog.ErrorDepthf(logger.depth, format, args...)
}

func (logger *GlogWrapper) Exitf(format string, args ...any) {
	glog.ExitDepthf(logger.depth, format, args...)
}

func (logger *GlogWrapper) Fatal(args ...any) {
	glog.FatalDepth(logger.depth, args...)
}

func (logger *GlogWrapper) Fatalf(format string, args ...any) {
	glog.FatalDepthf(logger.depth, format, args...)
}

var logInstance = &GlogWrapper{1}

func ProvideLogger() interfaces.ILogger {
	return logInstance
}
