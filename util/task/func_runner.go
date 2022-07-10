package task

import "time"

type funcRunner struct {
	run func() error
}

func (r funcRunner) Run() error {
	return r.run()
}

func NewTickerTaskFromFunc(interval time.Duration, runner func() error) *TickerTask {
	return NewTickerTask(interval, funcRunner{run: runner})
}
