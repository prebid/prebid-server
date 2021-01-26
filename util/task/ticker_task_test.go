package task_test

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/util/task"
	"github.com/stretchr/testify/assert"
)

type MockRunner struct {
	RunCount int
}

func (mcc *MockRunner) Run() error {
	mcc.RunCount++
	return nil
}

func TestStartWithSingleRun(t *testing.T) {
	// Setup:
	runner := &MockRunner{RunCount: 0}
	interval := 0 * time.Millisecond
	ticker := task.NewTickerTask(interval, runner)

	// Execute:
	ticker.Start()
	time.Sleep(10 * time.Millisecond)

	// Verify:
	assert.Equal(t, runner.RunCount, 1, "runner should have run one time")
}

func TestStartWithPeriodicRun(t *testing.T) {
	// Setup:
	runner := &MockRunner{RunCount: 0}
	interval := 10 * time.Millisecond
	ticker := task.NewTickerTask(interval, runner)

	// Execute:
	ticker.Start()
	time.Sleep(25 * time.Millisecond)
	ticker.Stop()

	// Verify:
	assert.Equal(t, runner.RunCount, 3, "runner should have run three times")
}

func TestStop(t *testing.T) {
	// Setup:
	runner := &MockRunner{RunCount: 0}
	interval := 10 * time.Millisecond
	ticker := task.NewTickerTask(interval, runner)

	// Execute:
	ticker.Start()
	time.Sleep(25 * time.Millisecond)
	ticker.Stop()
	time.Sleep(25 * time.Millisecond) // wait in case stop failed so additional runs can happen

	// Verify:
	assert.Equal(t, runner.RunCount, 3, "runner should have run three times")
}
