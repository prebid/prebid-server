package task_test

import (
	"sync"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/util/task"
	"github.com/stretchr/testify/assert"
)

type MockRunner struct {
	ExpectationMet chan struct{}
	actualCalls    int
	expectedCalls  int
	mutex          sync.Mutex
}

func NewMockRunner(expectedCalls int) *MockRunner {
	return &MockRunner{
		ExpectationMet: make(chan struct{}),
		expectedCalls:  expectedCalls,
	}
}

func (m *MockRunner) Run() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.actualCalls++

	if m.expectedCalls == m.actualCalls {
		close(m.ExpectationMet)
	}

	return nil
}

func (m *MockRunner) RunCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.actualCalls
}

func TestStartWithSingleRun(t *testing.T) {
	// Setup Initial Run Only:
	expectedRuns := 1
	runner := NewMockRunner(expectedRuns)
	interval := 0 * time.Millisecond // forces a single run
	ticker := task.NewTickerTask(interval, runner)

	// Execute:
	ticker.Start()

	// Verify:
	select {
	case <-runner.ExpectationMet:
	case <-time.After(250 * time.Millisecond):
		assert.Failf(t, "Runner Calls", "expected %v calls, observed %v calls", expectedRuns, runner.RunCount())
	}

	// Verify No Additional Runs:
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, expectedRuns, runner.RunCount(), "runner should not run after Stop is called")
}

func TestStartWithPeriodicRun(t *testing.T) {
	// Setup Initial Run + One Periodic Run:
	expectedRuns := 2
	runner := NewMockRunner(expectedRuns)
	interval := 10 * time.Millisecond
	ticker := task.NewTickerTask(interval, runner)

	// Execute:
	ticker.Start()

	// Verify Expected Runs:
	select {
	case <-runner.ExpectationMet:
		ticker.Stop()
	case <-time.After(250 * time.Millisecond):
		assert.Failf(t, "Runner Calls", "expected %v calls, observed %v calls", expectedRuns, runner.RunCount())
	}

	// Verify No Additional Runs After Stop:
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, expectedRuns, runner.RunCount(), "runner should not run after Stop is called")
}
