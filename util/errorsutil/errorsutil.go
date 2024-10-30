package errorsutil

import (
	"fmt"
	"runtime/debug"

	"github.com/golang/glog"
)

func LogPanic(r any) error {
	var err error
	if rErr, ok := r.(error); ok {
		err = rErr
	} else {
		err = fmt.Errorf("panic: %v", r)
	}
	glog.Error(err)
	debug.PrintStack()

	return err
}
