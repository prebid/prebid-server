// package tasks
package currencies

import (
	"time"
)

type Runner interface { // TODO(bfs): Runner? Runnable?
	Run() error
	GetRunNotifier() chan<- int
}

type TickerTask struct {
	fetchingInterval time.Duration
	taskRunner       Runner
	done             chan bool
}

func NewTickerTask(fetchingInterval time.Duration, taskRunner Runner) TickerTask {
	return TickerTask{
		fetchingInterval: fetchingInterval,
		taskRunner:       taskRunner,
	}
}

// Start begins periodic fetching at the given interval
// It triggers a run before beginning the timer if specified
// It returns a chan in which the number of data updates everytime a new update was done
func (tt *TickerTask) Start(runOnStart bool) {
	if runOnStart == true {
		tt.taskRunner.Run()
	}

	ticker := time.NewTicker(tt.fetchingInterval)
	ticksCount := 0

	for {
		select {
		case <-ticker.C:
			// Retries are handled by clients directly
			tt.taskRunner.Run()
			ticksCount++

			if runNotifier := tt.taskRunner.GetRunNotifier(); runNotifier != nil {
				runNotifier <- ticksCount
			}
		case <-tt.done:
			if ticker != nil {
				ticker.Stop()
				ticker = nil
			}
			return
		}
	}
}

// Stop stops periodic task but the task runner maintains state
func (tt *TickerTask) Stop() {
	tt.done <- true
	close(tt.done)
}
