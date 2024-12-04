package adscert

import (
	"fmt"
	"github.com/prebid/prebid-server/v3/di"
)

type SignerLogger struct {
}

func (sl *SignerLogger) Debugf(format string, args ...interface{}) {
	//there is no Debug level in glog
	di.Log.Infof(format, args...)
}

func (sl *SignerLogger) Infof(format string, args ...interface{}) {
	di.Log.Infof(format, args...)
}

func (sl *SignerLogger) Info(format string) {
	di.Log.Info(format)
}

func (sl *SignerLogger) Warningf(format string, args ...interface{}) {
	di.Log.Warningf(format, args...)
}

func (sl *SignerLogger) Errorf(format string, args ...interface{}) {
	di.Log.Errorf(format, args...)
}

func (sl *SignerLogger) Fatalf(format string, args ...interface{}) {
	di.Log.Fatalf(format, args...)
}

func (sl *SignerLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
