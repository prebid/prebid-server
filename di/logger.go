package di

import (
	"github.com/golang/glog"
)

type ILogger interface {
	Warningf(format string, args ...interface{})
}

type DefaultLogger struct{}

func (d *DefaultLogger) Warningf(format string, args ...interface{}) {
	glog.Warningf(format, args...)
}

var defaultLogger = DefaultLogger{}
var Logger ILogger = &defaultLogger
