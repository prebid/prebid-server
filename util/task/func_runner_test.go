package task

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTickerTaskFromFunc(t *testing.T) {
	var runCountMutex sync.Mutex
	runCount := 0

	funcTest := func() error {
		runCountMutex.Lock()
		defer runCountMutex.Unlock()
		runCount++
		return nil
	}

	anyDuration := 1 * time.Hour // not used for this test
	task := NewTickerTaskFromFunc(anyDuration, funcTest)

	err := task.runner.Run()
	assert.NoError(t, err)
	assert.Equal(t, 1, runCount)
}
