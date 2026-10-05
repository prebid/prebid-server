package logger

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withCleanExitHooks gives a test a pristine, isolated exit-hook registry. The
// registry is process-global and runs at most once, so without this a test would
// both inherit hooks from earlier tests and be unable to run them at all. The
// cleanup also drains any run still in flight, so a runner goroutine cannot
// outlive the test and report into the next one's mock.
func withCleanExitHooks(t *testing.T) {
	t.Helper()
	resetExitHooksForTest(t)
	t.Cleanup(func() { resetExitHooksForTest(t) })
}

func TestRunExitHooks_RunsInReverseRegistrationOrder(t *testing.T) {
	withCleanExitHooks(t)

	var mu sync.Mutex
	var order []string
	for _, name := range []string{"first", "second", "third"} {
		RegisterExitHook(name, func(context.Context) error {
			// Guarded because the runner is a separate goroutine: on the happy
			// path RunExitHooks returning is a happens-before edge, but if the
			// budget ever blew there would be none.
			mu.Lock()
			defer mu.Unlock()
			order = append(order, name)
			return nil
		})
	}

	require.NoError(t, RunExitHooks(context.Background()))

	// LIFO, mirroring defer: what was set up last is torn down first.
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"third", "second", "first"}, order)
}

func TestRunExitHooks_RunsAtMostOnce(t *testing.T) {
	withCleanExitHooks(t)

	var calls atomic.Int64
	RegisterExitHook("counter", func(context.Context) error {
		calls.Add(1)
		return nil
	})

	for range 3 {
		require.NoError(t, RunExitHooks(context.Background()))
	}

	assert.EqualValues(t, 1, calls.Load(), "exit hooks must not run again on a second shutdown path")
}

// TestRunExitHooks_SecondCallerWaitsForTheRunInFlight is the guard on the
// scenario the registry exists to survive: two goroutines reach a fatal at once.
// If the second caller returned immediately it would race ahead to os.Exit and
// truncate the flush the first one started.
func TestRunExitHooks_SecondCallerWaitsForTheRunInFlight(t *testing.T) {
	withCleanExitHooks(t)

	flushing := make(chan struct{})
	var flushed atomic.Bool
	RegisterExitHook("slow-flush", func(context.Context) error {
		close(flushing)
		time.Sleep(50 * time.Millisecond)
		flushed.Store(true)
		return nil
	})

	go func() { _ = RunExitHooks(context.Background()) }()
	<-flushing

	require.NoError(t, RunExitHooks(context.Background()))
	assert.True(t, flushed.Load(), "the second caller must not return while a flush is still in flight")
}

// TestRunExitHooks_SecondCallerGivesUpAtItsOwnBudget is the other half of that
// guarantee: waiting for someone else's flush must not become an unbounded stall
// on the fatal path.
func TestRunExitHooks_SecondCallerGivesUpAtItsOwnBudget(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	started := make(chan struct{})
	RegisterExitHook("hangs", func(context.Context) error {
		close(started)
		<-release
		return nil
	})

	go func() { _ = RunExitHooks(context.Background()) }()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := RunExitHooks(ctx)

	assert.Less(t, time.Since(start), 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gave up waiting for exit hooks started elsewhere")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRunExitHooks_ReentrantCallDoesNotDeadlock(t *testing.T) {
	withCleanExitHooks(t)

	// A hook that trips a second fatal path (directly, or via a library that
	// does) must not block waiting on the shutdown it is itself part of.
	var reentered atomic.Bool
	RegisterExitHook("reentrant", func(ctx context.Context) error {
		require.NoError(t, RunExitHooks(ctx))
		reentered.Store(true)
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = RunExitHooks(context.Background())
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunExitHooks deadlocked on a reentrant call from inside a hook")
	}
	assert.True(t, reentered.Load(), "the reentrant call should return immediately, not hang")
}

func TestRunExitHooks_PanickingHookDoesNotStopTheRest(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	var survivorRan atomic.Bool
	RegisterExitHook("survivor", func(context.Context) error {
		survivorRan.Store(true)
		return nil
	})
	RegisterExitHook("panicker", func(context.Context) error {
		panic("boom")
	})

	var err error
	require.NotPanics(t, func() { err = RunExitHooks(context.Background()) })

	assert.True(t, survivorRan.Load(), "a panicking hook must not prevent the remaining hooks from running")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `exit hook "panicker" failed`)
	assert.Contains(t, err.Error(), "panic: boom")
	assert.Contains(t, err.Error(), "runExitHook", "the recovered panic should carry its stack")

	require.Len(t, errorfMessages(mock), 1, "the recovered panic should also be reported")
	assert.Contains(t, errorfMessages(mock)[0], "panicker")
}

func TestRunExitHooks_FailingHookIsReportedAndTheRestStillRun(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	wantErr := errors.New("flush failed")
	var survivorRan atomic.Bool
	RegisterExitHook("survivor", func(context.Context) error {
		survivorRan.Store(true)
		return nil
	})
	RegisterExitHook("failer", func(context.Context) error {
		return wantErr
	})

	err := RunExitHooks(context.Background())

	assert.True(t, survivorRan.Load(), "a failing hook must not prevent the remaining hooks from running")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr, "the hook's error should be returned to the caller")
	assert.Contains(t, err.Error(), `exit hook "failer" failed`)

	require.Len(t, errorfMessages(mock), 1, "the failure should also be logged")
	assert.Contains(t, errorfMessages(mock)[0], "failer")
}

func TestRunExitHooks_HangingHookIsAbandonedAtTheBudget(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	// Released only in cleanup, so the hook is still parked when the budget
	// expires — the exact situation the budget exists to survive. Cleanups run
	// LIFO, so this releases before withCleanExitHooks drains the runner.
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	var laterHookRan atomic.Bool
	RegisterExitHook("never-reached", func(context.Context) error {
		laterHookRan.Store(true)
		return nil
	})
	RegisterExitHook("hangs", func(context.Context) error {
		<-release
		return nil
	})

	SetExitHookTimeout(50 * time.Millisecond)

	start := time.Now()
	err := RunExitHooks(context.Background())
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 5*time.Second, "a hanging hook must not hold up termination")
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "the full budget should be given before abandoning")
	assert.False(t, laterHookRan.Load(), "hooks queued behind an exhausted budget should not run")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `exit hook "hangs" abandoned`, "the report should name the hook that hung")
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	require.Len(t, errorfMessages(mock), 1)
	assert.Contains(t, errorfMessages(mock)[0], "hangs")
}

// TestRunExitHooks_HookHonoringItsBudgetIsNotReportedAsHung covers the case a
// naive select gets wrong: a hook that waits on ctx.Done and returns cleanly
// finishes microseconds after the deadline that also wakes the waiter, so
// without a drain grace period every well-behaved hook is blamed for hanging.
func TestRunExitHooks_HookHonoringItsBudgetIsNotReportedAsHung(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	RegisterExitHook("polite", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	SetExitHookTimeout(20 * time.Millisecond)

	assert.NoError(t, RunExitHooks(context.Background()),
		"a hook that honors its budget has not failed")
	assert.Empty(t, errorfMessages(mock), "nothing should be reported as abandoned")
}

// TestRunExitHooks_SkippedHooksAreReportedEvenWhenTheRunningHookSucceeds guards
// the quietest failure mode: the hook in flight honors the budget and returns
// nil, so joining only the hook errors would hand the caller a clean nil while
// everything queued behind it was silently dropped.
func TestRunExitHooks_SkippedHooksAreReportedEvenWhenTheRunningHookSucceeds(t *testing.T) {
	withCleanExitHooks(t)

	var skippedRan atomic.Bool
	RegisterExitHook("skipped", func(context.Context) error {
		skippedRan.Store(true)
		return nil
	})
	RegisterExitHook("polite", func(ctx context.Context) error {
		<-ctx.Done()
		return nil // honors the budget, reports no error of its own
	})
	SetExitHookTimeout(20 * time.Millisecond)

	err := RunExitHooks(context.Background())

	assert.False(t, skippedRan.Load())
	require.Error(t, err, "skipping a hook is a failure of the run, not a quiet outcome")
	assert.Contains(t, err.Error(), "1 exit hooks skipped")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestRunExitHooks_AlreadyExpiredContextIsReportedHonestly covers a SIGTERM
// handler that reuses a shutdown context whose deadline has already passed: no
// hook can run, and the report must say so rather than naming a hook.
func TestRunExitHooks_AlreadyExpiredContextIsReportedHonestly(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	var ran atomic.Bool
	RegisterExitHook("never-runs", func(context.Context) error {
		ran.Store(true)
		return nil
	})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := RunExitHooks(ctx)

	assert.False(t, ran.Load())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup budget already exhausted")
	assert.Contains(t, err.Error(), "1 exit hooks skipped")
	require.Len(t, errorfMessages(mock), 1)

	// Crucially the registry stays open: a stale context handed to a defensive
	// shutdown call must not cost the process the cleanup a later fatal needs.
	require.NoError(t, RunExitHooks(context.Background()))
	assert.True(t, ran.Load(), "a healthy shutdown path after an expired one should still run the hooks")
}

func TestRunExitHooks_CallerDeadlineTakesPrecedenceOverTheDefaultBudget(t *testing.T) {
	withCleanExitHooks(t)
	SetExitHookTimeout(time.Hour) // would dominate if the caller's deadline were ignored

	var deadline atomic.Pointer[time.Time]
	RegisterExitHook("inspect", func(ctx context.Context) error {
		if d, ok := ctx.Deadline(); ok {
			deadline.Store(&d)
		}
		return nil
	})

	want := time.Now().Add(30 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), want)
	defer cancel()
	require.NoError(t, RunExitHooks(ctx))

	got := deadline.Load()
	require.NotNil(t, got, "hooks should receive the caller's deadline")
	assert.WithinDuration(t, want, *got, time.Second)
}

func TestRunExitHooks_AppliesTheDefaultBudgetWhenTheCallerHasNoDeadline(t *testing.T) {
	withCleanExitHooks(t)

	var hadDeadline atomic.Bool
	RegisterExitHook("inspect", func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		hadDeadline.Store(ok)
		return nil
	})

	require.NoError(t, RunExitHooks(context.Background()))

	assert.True(t, hadDeadline.Load(), "hooks should be bounded even when the caller passes an unbounded context")
}

// TestSetExitHookTimeout_NonPositiveRestoresTheDefault matters because zero is
// what an unset config field yields: SetExitHookTimeout(cfg.ShutdownTimeout)
// must not silently turn a bounded fatal path into one that can block forever.
func TestSetExitHookTimeout_NonPositiveRestoresTheDefault(t *testing.T) {
	withCleanExitHooks(t)

	for _, d := range []time.Duration{0, -time.Second} {
		SetExitHookTimeout(time.Millisecond)
		SetExitHookTimeout(d)

		exitHookMu.Lock()
		got := exitHookTimeout
		exitHookMu.Unlock()
		assert.Equalf(t, DefaultExitHookTimeout, got, "SetExitHookTimeout(%v) should restore the default", d)
	}
}

// TestRunExitHooks_WithNoHooksLeavesTheRegistryOpen protects a late-initialising
// subsystem: a defensive RunExitHooks before anything has registered must not
// permanently disable cleanup for the process.
func TestRunExitHooks_WithNoHooksLeavesTheRegistryOpen(t *testing.T) {
	withCleanExitHooks(t)

	require.NoError(t, RunExitHooks(context.Background()))

	var ran atomic.Bool
	RegisterExitHook("registered-later", func(context.Context) error {
		ran.Store(true)
		return nil
	})
	require.NoError(t, RunExitHooks(context.Background()))

	assert.True(t, ran.Load(), "a run with nothing registered should not close the registry")
}

func TestRegisterExitHook_IgnoresNil(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	RegisterExitHook("nil-hook", nil)

	assert.NotPanics(t, func() { _ = RunExitHooks(context.Background()) },
		"a nil hook must be rejected at registration, not panic at shutdown")
	require.Len(t, errorfMessages(mock), 1)
	assert.Contains(t, errorfMessages(mock)[0], "nil-hook")
}

func TestRegisterExitHook_AfterShutdownBeganIsReportedAndSkipped(t *testing.T) {
	withCleanExitHooks(t)
	mock := newMockLogger()
	swapLogger(t, mock)

	var lateRan atomic.Bool
	RegisterExitHook("early", func(context.Context) error {
		RegisterExitHook("late", func(context.Context) error {
			lateRan.Store(true)
			return nil
		})
		return nil
	})

	require.NoError(t, RunExitHooks(context.Background()))

	assert.False(t, lateRan.Load(), "hooks registered once shutdown began must not run")

	exitHookMu.Lock()
	registered := len(exitHooks)
	exitHookMu.Unlock()
	assert.Equal(t, 1, registered, "a late hook must be refused, not appended and silently skipped")

	require.Len(t, warnfMessages(mock), 1, "a late registration should be reported")
	assert.Contains(t, warnfMessages(mock)[0], "late")
}

// TestRegisterExitHook_IsSafeForConcurrentUse exercises the registry lock; the
// race detector is the assertion.
func TestRegisterExitHook_IsSafeForConcurrentUse(t *testing.T) {
	withCleanExitHooks(t)

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RegisterExitHook("concurrent", func(context.Context) error { return nil })
			if i%8 == 0 {
				SetExitHookTimeout(time.Second)
			}
		}()
	}
	wg.Wait()

	exitHookMu.Lock()
	registered := len(exitHooks)
	exitHookMu.Unlock()
	assert.Equal(t, 32, registered)
}
