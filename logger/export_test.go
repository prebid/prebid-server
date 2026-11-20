package logger

import (
	"testing"
	"time"
)

// resetExitHooksForTest returns the process-global exit-hook registry to its
// initial state so each test starts clean. The registry is deliberately
// write-once-then-run in production, which is why this lives in a _test file and
// never ships in the binary.
//
// A run still in flight is drained first. The runner goroutine holds a reference
// to the Logger the test installed and writes to glog, while the test may be
// swapping os.Stderr; letting it outlive the test turns that into a data race
// with a baffling stack, so a drain that does not finish is a test failure, not
// something to shrug off.
func resetExitHooksForTest(t *testing.T) {
	t.Helper()

	exitHookMu.Lock()
	run := exitHookCurrent
	exitHookMu.Unlock()

	if run != nil {
		select {
		case <-run.done:
		case <-time.After(2 * time.Second):
			t.Errorf("exit hooks did not finish; a runner goroutine is leaking into later tests")
		}
	}

	exitHookMu.Lock()
	defer exitHookMu.Unlock()
	exitHooks = nil
	exitHookTimeout = DefaultExitHookTimeout
	exitHookCurrent = nil
}
