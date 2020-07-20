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
func (tt *TickerTask) Start() {
	if tt.interval <= 0 {
		return
	}

	tt.runner.Run()

	go tt.run()
}

// Stop stops the periodic task but the task runner maintains state
func (tt *TickerTask) Stop() {
	tt.done <- true
	close(tt.done)
}

// run creates a ticker that ticks at the specified interval. On each tick,
// the task is executed and the runner is notified
func (tt *TickerTask) run() {
	ticker := time.NewTicker(tt.interval)

	for {
		select {
		case <-ticker.C:
			tt.runner.Run()
			tt.runner.Notify()
		case <-tt.done:
			if ticker != nil {
				ticker.Stop()
			}
			return
		}
	}
}
