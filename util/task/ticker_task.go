package task

import (
	"time"
)

type Runner interface {
	Run() error
}

// runner func
type funcRunner struct {
	run func() error
}

func (r funcRunner) Run() error {
	return r.run()
}

// end runner func

type TickerTask struct {
	interval time.Duration
	runner   Runner
	done     chan struct{}
}

func NewTickerTaskFromFunc(interval time.Duration, runner func() error) *TickerTask {
	return NewTickerTask(interval, funcRunner{run: runner})
}

func NewTickerTask(interval time.Duration, runner Runner) *TickerTask {
	return &TickerTask{
		interval: interval,
		runner:   runner,
		done:     make(chan struct{}),
	}
}

// Start runs the task immediately and then schedules the task to run periodically
// if a positive fetching interval has been specified.
func (t *TickerTask) Start() {
	t.runner.Run()

	if t.interval > 0 {
		go t.runRecurring()
	}
}

// Stop stops the periodic task but the task runner maintains state
func (t *TickerTask) Stop() {
	close(t.done)
}

// run creates a ticker that ticks at the specified interval. On each tick,
// the task is executed
func (t *TickerTask) runRecurring() {
	ticker := time.NewTicker(t.interval)

	for {
		select {
		case <-ticker.C:
			t.runner.Run()
		case <-t.done:
			return
		}
	}
}
