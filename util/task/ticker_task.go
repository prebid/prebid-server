package task

import (
	"time"
)

type Runner interface {
	Run() error
}

type TickerTask struct {
	interval       time.Duration
	runner         Runner
	skipInitialRun bool
	done           chan struct{}
}

func NewTickerTask(interval time.Duration, runner Runner) *TickerTask {
	return NewTickerTaskWithOptions(Options{
		Interval: interval,
		Runner:   runner,
	})
}

type Options struct {
	Interval       time.Duration
	Runner         Runner
	SkipInitialRun bool
}

func NewTickerTaskWithOptions(opt Options) *TickerTask {
	return &TickerTask{
		interval:       opt.Interval,
		runner:         opt.Runner,
		skipInitialRun: opt.SkipInitialRun,
		done:           make(chan struct{}),
	}
}

// Start runs the task immediately and then schedules the task to run periodically
// if a positive fetching interval has been specified.
func (t *TickerTask) Start() {
	if !t.skipInitialRun {
		t.runner.Run()
	}

	if t.interval > 0 {
		go t.runRecurring()
	}
}

// Stop stops the periodic task but the task runner maintains state
func (t *TickerTask) Stop() {
	close(t.done)
}

// Done exports readonly done channel
func (t *TickerTask) Done() <-chan struct{} {
	return t.done
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
