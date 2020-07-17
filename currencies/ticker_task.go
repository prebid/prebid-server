package currencies

import (
	"time"
)

type Runner interface {
	Run() error
	GetRunNotifier() chan<- int
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
// the task is executed.
// It returns a chan which receives the number of times the task has run since it was last started.
func (tt *TickerTask) run() {
	ticker := time.NewTicker(tt.interval)
	ticksCount := 0

	for {
		select {
		case <-ticker.C:
			tt.runner.Run()
			ticksCount++

			if runNotifier := tt.runner.GetRunNotifier(); runNotifier != nil {
				runNotifier <- ticksCount
			}
		case <-tt.done:
			if ticker != nil {
				ticker.Stop()
			}
			return
		}
	}
}
