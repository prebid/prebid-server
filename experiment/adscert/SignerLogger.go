package adscert

import (
	"fmt"
	"github.com/prebid/prebid-server/v3/logger"
)

type SignerLogger struct {
}

func (sl *SignerLogger) Debugf(format string, args ...interface{}) {
	//there is no Debug level in glog
	logger.Log.Infof(format, args...)
}

func (sl *SignerLogger) Infof(format string, args ...interface{}) {
	logger.Log.Infof(format, args...)
}

func (sl *SignerLogger) Info(format string) {
	logger.Log.Info(format)
}

func (sl *SignerLogger) Warningf(format string, args ...interface{}) {
	logger.Log.Warningf(format, args...)
}

func (sl *SignerLogger) Errorf(format string, args ...interface{}) {
	logger.Log.Errorf(format, args...)
}

func (sl *SignerLogger) Fatalf(format string, args ...interface{}) {
	logger.Log.Fatalf(format, args...)
}

func (sl *SignerLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
