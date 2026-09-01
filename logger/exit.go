package logger

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// DefaultExitHookTimeout is the total budget RunExitHooks gives all registered
// hooks when the supplied context carries no deadline of its own. It bounds how
// long a fatal error can be delayed by cleanup: a hook that hangs must not keep
// the process alive, and must not stop the remaining shutdown steps from running.
//
// The worst case is this budget plus exitHookDrainGrace, since a hook that has
// blown the budget is given a moment to unwind before it is abandoned.
const DefaultExitHookTimeout = 5 * time.Second

// exitHookDrainGrace is how long RunExitHooks waits for the hook runner to
// unwind after the budget expires, before reporting a hook as abandoned. A hook
// that honors its context finishes within microseconds of the deadline it was
// given; without this grace period every well-behaved hook would be reported as
// hung, because the waiting goroutine is woken by the same timer.
const exitHookDrainGrace = 100 * time.Millisecond

// ExitHook is a cleanup function run before the process terminates.
//
// The signature deliberately matches the Shutdown/ForceFlush methods exposed by
// OpenTelemetry providers and by most buffered sinks, so they can be registered
// directly without an adapter:
//
//	logger.RegisterExitHook("otel-traces", tracerProvider.Shutdown)
//
// ctx carries the remaining cleanup budget, so a hook that can block should
// honor it. Returning an error is not fatal: the error is logged, reported in
// RunExitHooks' return value, and the remaining hooks still run.
//
// A hook runs on the fatal path, in a process that may already be unhealthy, and
// ahead of the message explaining why it is dying. Keep hooks short, allocation-
// light, and free of anything that can wedge; never call os.Exit from one.
type ExitHook func(ctx context.Context) error

type registeredExitHook struct {
	name string
	hook ExitHook
}

// exitHookRun is one execution of the registry. err is written by the runner
// goroutine before done is closed, so anything that has received from done may
// read it.
type exitHookRun struct {
	done chan struct{}
	err  error
}

// exitHooksRunningKey marks the context handed to a hook, so that a hook which
// re-enters RunExitHooks — directly, or through a library that logs a fatal —
// is recognised and returns immediately instead of waiting on the run it is
// itself part of.
type exitHooksRunningKey struct{}

// All of these are guarded by exitHookMu, which is never held while a hook or a
// Logger method runs, so a hook is free to call back into this package.
var (
	exitHookMu      sync.Mutex
	exitHooks       []registeredExitHook
	exitHookTimeout = DefaultExitHookTimeout
	exitHookCurrent *exitHookRun // non-nil once a run has begun
)

// RegisterExitHook registers a named cleanup function to run before the process
// terminates through any Fatal path (Fatalf, Fatal, FatalContext), or when
// RunExitHooks is called explicitly — for example from a SIGTERM handler.
//
// name identifies the hook in the diagnostics emitted when it fails or exceeds
// the cleanup budget; it need not be unique. A nil hook is ignored.
//
// Hooks run in reverse registration order (LIFO), mirroring defer: whatever was
// set up last is torn down first. Registration is safe for concurrent use.
//
// Hooks are meant to be registered during startup and live for the lifetime of
// the process; there is no way to remove one, and the registry holds a reference
// to every hook it is given. Registering after a run has begun is a no-op — the
// set of hooks is snapshotted when the run starts — and is reported as a warning.
func RegisterExitHook(name string, hook ExitHook) {
	if hook == nil {
		Errorf("logger: ignoring nil exit hook %q", name)
		return
	}

	exitHookMu.Lock()
	late := exitHookCurrent != nil
	if !late {
		exitHooks = append(exitHooks, registeredExitHook{name: name, hook: hook})
	}
	exitHookMu.Unlock()

	// Reported outside the lock: a Logger implementation is arbitrary code and
	// must not run while the registry is locked.
	if late {
		Warnf("logger: exit hook %q registered after shutdown began; it will not run", name)
	}
}

// SetExitHookTimeout overrides the total budget given to all exit hooks when
// RunExitHooks is called with a context that has no deadline of its own. A
// non-positive duration restores DefaultExitHookTimeout: there is deliberately
// no way to make the fatal path wait forever, since the zero value of
// time.Duration is exactly what an unset config field yields.
//
// It has no effect on a run that has already begun.
func SetExitHookTimeout(d time.Duration) {
	if d <= 0 {
		d = DefaultExitHookTimeout
	}
	exitHookMu.Lock()
	defer exitHookMu.Unlock()
	exitHookTimeout = d
}

// RunExitHooks runs every registered exit hook, most recently registered first,
// and returns once they have all finished or the cleanup budget is exhausted.
// It returns nil when every hook succeeded, and otherwise an error joining the
// hook failures with any budget exhaustion. Failures are also logged as they
// happen, because the caller on the fatal path is about to disappear.
//
// The hooks run at most once per process. A second caller — a SIGTERM handler
// racing a fatal, or a second goroutine hitting a fatal — waits for the run
// already in flight rather than racing ahead to os.Exit and truncating its
// flush, bounded by that caller's own budget. A hook that re-enters RunExitHooks
// with the context it was handed is recognised and returns immediately; one that
// re-enters through a fresh context (a hook that calls Fatalf, say) cannot be
// told apart from a genuine second caller, so it takes the bounded wait instead
// of returning at once. Either way it cannot deadlock.
//
// If ctx has no deadline, the timeout set by SetExitHookTimeout (default
// DefaultExitHookTimeout) is applied. The budget covers all hooks together, not
// each one: when it expires RunExitHooks returns, logging which hook was in
// flight, and any hook still running is abandoned. That is deliberate — on the
// fatal path the caller is about to terminate, and a stuck hook must not stop it.
//
// A hook that panics is recovered and reported, with its stack, like a returned
// error; the remaining hooks still run. If no hooks are registered the call does
// nothing and does not close the registry, so a subsystem that registers late
// still gets its cleanup.
func RunExitHooks(ctx context.Context) error {
	if ctx.Value(exitHooksRunningKey{}) != nil {
		return nil
	}

	// A context that is already done cannot run anything, so bail out before
	// claiming the registry: closing it here would leave a "completed" run that
	// flushed nothing, and every later shutdown path would inherit that verdict.
	if err := ctx.Err(); err != nil {
		exitHookMu.Lock()
		skipped, out := len(exitHooks), logger
		exitHookMu.Unlock()

		err = fmt.Errorf("logger: cleanup budget already exhausted, %d exit hooks skipped: %w", skipped, err)
		report(out, err)
		return err
	}

	exitHookMu.Lock()
	if run := exitHookCurrent; run != nil {
		exitHookMu.Unlock()
		return waitForExitHooks(ctx, run)
	}
	if len(exitHooks) == 0 {
		exitHookMu.Unlock()
		return nil
	}
	hooks := make([]registeredExitHook, len(exitHooks))
	copy(hooks, exitHooks)
	timeout := exitHookTimeout
	// Snapshot the Logger: the runner goroutine can outlive this call, and the
	// package-level logger var is not synchronised.
	out := logger
	run := &exitHookRun{done: make(chan struct{})}
	exitHookCurrent = run
	exitHookMu.Unlock()

	ctx, cancel := withExitHookBudget(ctx, timeout)
	defer cancel()

	// inFlight names the hook currently running so that, if the budget expires,
	// the report can say which one hung. It is written only by the runner and
	// read only after the budget expires, both under exitHookMu.
	var inFlight string
	hookCtx := context.WithValue(ctx, exitHooksRunningKey{}, struct{}{})

	go func() {
		var errs []error
		for i := len(hooks) - 1; i >= 0; i-- {
			if err := ctx.Err(); err != nil {
				// Skipping hooks is a failure of the run, not a quiet outcome:
				// without this the caller could be told everything flushed when
				// most of it never got the chance.
				errs = append(errs, fmt.Errorf(
					"logger: cleanup budget exhausted, %d exit hooks skipped: %w", i+1, err))
				break
			}

			exitHookMu.Lock()
			inFlight = hooks[i].name
			exitHookMu.Unlock()

			err := runExitHook(hookCtx, hooks[i].hook)

			// Cleared before reporting, so a hook that has already returned is
			// never named as the one that hung.
			exitHookMu.Lock()
			inFlight = ""
			exitHookMu.Unlock()

			if err != nil {
				err = fmt.Errorf("logger: exit hook %q failed: %w", hooks[i].name, err)
				report(out, err)
				errs = append(errs, err)
			}
		}
		run.err = errors.Join(errs...)
		close(run.done)
	}()

	select {
	case <-run.done:
		return run.err
	case <-ctx.Done():
	}

	// The budget expired. A hook that honored its context is unwinding right
	// now, so wait a moment before accusing it of hanging.
	drain := time.NewTimer(exitHookDrainGrace)
	defer drain.Stop()
	select {
	case <-run.done:
		return run.err
	case <-drain.C:
	}

	exitHookMu.Lock()
	name := inFlight
	exitHookMu.Unlock()

	var err error
	if name == "" {
		err = fmt.Errorf("logger: cleanup budget exhausted with no exit hook in flight: %w", ctx.Err())
	} else {
		err = fmt.Errorf("logger: exit hook %q abandoned, cleanup budget exhausted: %w", name, ctx.Err())
	}
	report(out, err)
	return err
}

// report logs a shutdown diagnostic through out.
//
// It exists as a call frame of its own so that the file:line glog records is the
// one in this file. The Logger implementations skip a fixed number of frames,
// sized for the package-level wrappers in logger.go; calling a Logger method
// directly from here would skip one frame too many and attribute every exit-hook
// diagnostic to whoever called RunExitHooks — or, from the runner goroutine, to
// runtime.goexit.
func report(out Logger, err error) {
	out.Errorf("%v", err)
}

// waitForExitHooks blocks until the run already in flight finishes or the
// caller's own budget runs out, so a second fatal cannot cut short the flush a
// first one started.
func waitForExitHooks(ctx context.Context, run *exitHookRun) error {
	ctx, cancel := withExitHookBudget(ctx, currentExitHookTimeout())
	defer cancel()

	select {
	case <-run.done:
		return run.err
	case <-ctx.Done():
		err := fmt.Errorf("logger: gave up waiting for exit hooks started elsewhere: %w", ctx.Err())
		report(logger, err)
		return err
	}
}

func currentExitHookTimeout() time.Duration {
	exitHookMu.Lock()
	defer exitHookMu.Unlock()
	return exitHookTimeout
}

// withExitHookBudget bounds ctx by timeout unless the caller already supplied a
// deadline of its own, which always wins.
func withExitHookBudget(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// runExitHook invokes a single hook, converting a panic into an error so that
// one misbehaving hook cannot take down the rest of the shutdown sequence. The
// stack is captured here because it is gone by the time the error is reported.
func runExitHook(ctx context.Context, hook ExitHook) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	return hook(ctx)
}
