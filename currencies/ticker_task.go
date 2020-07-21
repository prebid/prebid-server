package currencies

import (
	"time"
)

type Runner interface {
	Run() error
	Notify()
}

type TickerTask struct {
	interval time.Duration
	runner   Runner
	done     chan bool
}

func NewTickerTask(interval time.Duration, runner Runner) TickerTask {
	return TickerTask{
		interval: interval,
		runner:   runner,
		done:     make(chan bool),
	}
}

// Start runs the task immediately and then schedules the task to run periodically
// if a positive fetching interval has been specified.
func (t *TickerTask) Start() {
	if t.interval <= 0 {
		return
	}

	t.runner.Run()

	go t.run()
}

// Stop stops the periodic task but the task runner maintains state
func (t *TickerTask) Stop() {
	t.done <- true
	close(t.done)
}

// run creates a ticker that ticks at the specified interval. On each tick,
// the task is executed and the runner is notified
func (t *TickerTask) run() {
	ticker := time.NewTicker(t.interval)

	for {
		select {
		case <-ticker.C:
			t.runner.Run()
			t.runner.Notify()
		case <-t.done:
			if ticker != nil {
				ticker.Stop()
			}
			return
		}
	}
}
