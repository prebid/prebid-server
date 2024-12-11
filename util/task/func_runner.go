package task

import "time"

type FuncRunner struct {
	run func() error
}

func (r FuncRunner) Run() error {
	return r.run()
}

func NewTickerTaskFromFunc(interval time.Duration, runner func() error) *TickerTask {
	return NewTickerTask(interval, FuncRunner{run: runner})
}

func NewFuncRunner(f func() error) *FuncRunner {
	return &FuncRunner{run: f}
}
