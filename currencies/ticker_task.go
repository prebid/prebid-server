package currencies

import (
	"time"
)

type Runner interface {
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
		done:             make(chan bool),
	}
}

func (tt *TickerTask) Start(runOnStart bool) {
	// Only schedule periodic task if a fetching interval has been specified
	if tt.fetchingInterval <= 0 {
		return
	}

	if runOnStart == true {
		tt.taskRunner.Run()
	}

	go tt.run()
}

// Stop stops periodic task but the task runner maintains state
func (tt *TickerTask) Stop() {
	tt.done <- true
	close(tt.done)
}

// Start begins periodic fetching at the given interval
// It triggers a run before beginning the timer if specified
// It returns a chan in which the number of data updates everytime a new update was done
func (tt *TickerTask) run() {
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
