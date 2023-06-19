package adscert

import (
	"fmt"
	"github.com/golang/glog"
)

type SignerLogger struct {
}

func (sl *SignerLogger) Debugf(format string, args ...interface{}) {
	//there is no Debug level in glog
	glog.Infof(format, args...)
}

func (sl *SignerLogger) Infof(format string, args ...interface{}) {
	glog.Infof(format, args...)
}

func (sl *SignerLogger) Info(format string) {
	glog.Info(format)
}

func (sl *SignerLogger) Warningf(format string, args ...interface{}) {
	glog.Warningf(format, args...)
}

func (sl *SignerLogger) Errorf(format string, args ...interface{}) {
	glog.Errorf(format, args...)
}

func (sl *SignerLogger) Fatalf(format string, args ...interface{}) {
	glog.Fatalf(format, args...)
}

func (sl *SignerLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
