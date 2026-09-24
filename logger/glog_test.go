package logger

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCtxKey is a private context key type. A bare string would collide with any
// other package storing the same key, and trips staticcheck SA1029.
type testCtxKey string

const testRequestIDKey testCtxKey = "requestID"

// setGlogFlag sets a glog flag for the duration of a test and restores it
// afterwards. glog's flags are process-global, so a test that leaves one changed
// silently reconfigures everything that runs after it.
func setGlogFlag(t *testing.T, name, value string) {
	t.Helper()
	f := flag.Lookup(name)
	require.NotNilf(t, f, "glog flag %q should be registered", name)
	prev := f.Value.String()
	require.NoError(t, flag.Set(name, value))
	t.Cleanup(func() { _ = flag.Set(name, prev) })
}

// captureGlogStderr runs fn with glog's stderr sink pointed at a pipe, and
// returns everything glog wrote. It swaps process-global state (os.Stderr and
// two glog flags), so tests that use it must not call t.Parallel and must not
// leave a goroutine logging in the background. glog's stderr sink resolves os.Stderr on every
// write, so swapping it here is enough to capture real output — which is what
// makes it possible to assert on glog's actual line format, severity letter and
// file:line rather than merely that a call did not panic.
func captureGlogStderr(t *testing.T, fn func()) string {
	t.Helper()
	setGlogFlag(t, "logtostderr", "true")
	setGlogFlag(t, "stderrthreshold", "INFO")

	r, w, err := os.Pipe()
	require.NoError(t, err)

	prev := os.Stderr
	os.Stderr = w
	func() {
		defer func() { os.Stderr = prev }()
		fn()
		glog.Flush()
	}()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(out)
}

// currentLine reports the line number of its own call site.
func currentLine() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

// glogLinePattern matches glog's documented line prefix:
// Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg
func glogLinePattern(severity, file string, line int, msg string) string {
	return `^` + severity + `\d{4} \d{2}:\d{2}:\d{2}\.\d{6}\s+\d+ ` +
		file + `:` + strconv.Itoa(line) + `\] ` + msg + "\n$"
}

func TestNewGlogLogger(t *testing.T) {
	log := NewGlogLogger()

	require.NotNil(t, log, "NewGlogLogger should return a non-nil logger")
	_, ok := log.(*GlogLogger)
	assert.True(t, ok, "Logger should be of type *GlogLogger")
}

// TestNewGlogLogger_AttributesToItsDirectCaller is the contract for the exported
// constructor: a logger obtained from it reports the line that called it, with
// no wrapper assumed. The package-level functions in logger.go add a frame of
// their own and so use their own instance — mixing the two up is what put
// logger.go on every line prebid-server logged.
func TestNewGlogLogger_AttributesToItsDirectCaller(t *testing.T) {
	log := NewGlogLogger()

	formattedLine := currentLine() + 1
	formatted := captureGlogStderr(t, func() { log.Infof("direct formatted") })

	structuredLine := currentLine() + 1
	structured := captureGlogStderr(t, func() { log.Info("direct structured") })

	assert.Regexp(t, glogLinePattern("I", `glog_test\.go`, formattedLine, `direct formatted`), formatted)
	assert.Regexp(t, glogLinePattern("I", `glog_test\.go`, structuredLine, `direct structured`), structured)
}

// TestGlogLogger_StructuredOutputGoesThroughGlog is the contract test for the
// structured path. It pins, against glog's real output, that a structured record
// carries glog's line format, the right severity letter, and the *caller's*
// file:line — which is what proves structuredCallDepth is still correct. A
// handler that wrote its own lines to stderr would pass none of this.
func TestGlogLogger_StructuredOutputGoesThroughGlog(t *testing.T) {
	var wantLine int
	out := captureGlogStderr(t, func() {
		wantLine = currentLine() + 1
		Info("structured message", "key", "value", "count", 7)
	})

	assert.Regexp(t,
		glogLinePattern("I", `glog_test\.go`, wantLine, `structured message, key=value, count=7`),
		out)
}

// TestGlogLogger_StructuredSeverityRouting pins the slog level to glog severity
// mapping. Without it, a record logged at the wrong severity lands in the wrong
// log file and no test notices.
func TestGlogLogger_StructuredSeverityRouting(t *testing.T) {
	cases := []struct {
		name     string
		log      func(msg string)
		severity string
	}{
		// glog has no debug severity; Debug records go to INFO, matching Debugf.
		{"debug", func(msg string) { Debug(msg) }, "I"},
		{"info", func(msg string) { Info(msg) }, "I"},
		{"warn", func(msg string) { Warn(msg) }, "W"},
		{"error", func(msg string) { Error(msg) }, "E"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureGlogStderr(t, func() { tc.log("severity probe") })
			require.NotEmpty(t, out, "glog should have emitted a line")
			assert.Equalf(t, tc.severity, out[:1],
				"%s should be recorded at glog severity %s, got line %q", tc.name, tc.severity, out)
		})
	}
}

// TestGlogLogger_FormattedSeverityRouting is the same contract for the
// pre-existing printf-style methods, so the two paths are held to one standard.
func TestGlogLogger_FormattedSeverityRouting(t *testing.T) {
	cases := []struct {
		name     string
		log      func(msg string)
		severity string
	}{
		{"debugf", func(msg string) { Debugf("%s", msg) }, "I"},
		{"infof", func(msg string) { Infof("%s", msg) }, "I"},
		{"warnf", func(msg string) { Warnf("%s", msg) }, "W"},
		{"errorf", func(msg string) { Errorf("%s", msg) }, "E"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureGlogStderr(t, func() { tc.log("severity probe") })
			require.NotEmpty(t, out, "glog should have emitted a line")
			assert.Equalf(t, tc.severity, out[:1],
				"%s should be recorded at glog severity %s, got line %q", tc.name, tc.severity, out)
		})
	}
}

// TestGlogLogger_FormattedOutputAttribution pins the same contract for the
// printf-style methods. Before the call-depth fix these reported
// logger/logger.go — the package-level wrapper — for every line prebid-server
// logs, which is exactly the regression an assert.NotPanics test cannot see.
func TestGlogLogger_FormattedOutputAttribution(t *testing.T) {
	var wantLine int
	out := captureGlogStderr(t, func() {
		wantLine = currentLine() + 1
		Infof("formatted %s", "message")
	})

	assert.Regexp(t, glogLinePattern("I", `glog_test\.go`, wantLine, `formatted message`), out)
}

// TestPackageFunctionAttribution pins the call depth for every non-fatal
// package-level entry point. The two paths have different frame counts, and the
// Context variants take a different route again, so covering only Info and Infof
// would leave ten ways to be silently wrong. Each case logs and reports its own
// line from the same source line, so the expectation cannot drift.
func TestPackageFunctionAttribution(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		severity string
		log      func() int
	}{
		{"Debugf", "I", func() int { Debugf("probe"); return currentLine() }},
		{"Infof", "I", func() int { Infof("probe"); return currentLine() }},
		{"Warnf", "W", func() int { Warnf("probe"); return currentLine() }},
		{"Errorf", "E", func() int { Errorf("probe"); return currentLine() }},
		{"Debug", "I", func() int { Debug("probe"); return currentLine() }},
		{"Info", "I", func() int { Info("probe"); return currentLine() }},
		{"Warn", "W", func() int { Warn("probe"); return currentLine() }},
		{"Error", "E", func() int { Error("probe"); return currentLine() }},
		{"DebugContext", "I", func() int { DebugContext(ctx, "probe"); return currentLine() }},
		{"InfoContext", "I", func() int { InfoContext(ctx, "probe"); return currentLine() }},
		{"WarnContext", "W", func() int { WarnContext(ctx, "probe"); return currentLine() }},
		{"ErrorContext", "E", func() int { ErrorContext(ctx, "probe"); return currentLine() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var wantLine int
			out := captureGlogStderr(t, func() { wantLine = tc.log() })
			assert.Regexp(t, glogLinePattern(tc.severity, `glog_test\.go`, wantLine, "probe"), out)
		})
	}
}

func TestGlogLogger_ContextVariantsReachGlog(t *testing.T) {
	ctx := context.WithValue(context.Background(), testRequestIDKey, "abc123")

	out := captureGlogStderr(t, func() {
		InfoContext(ctx, "with context", "requestID", "abc123")
	})

	assert.Contains(t, out, "with context, requestID=abc123")
}

// TestGlogLogger_ZeroValueIsUsable guards the exported type's zero value. A
// GlogLogger built as a struct literal predates this package's constructor and
// must keep working; the structured methods build their handler on demand.
func TestGlogLogger_ZeroValueIsUsable(t *testing.T) {
	glogLogger := &GlogLogger{depth: 2}

	assert.NotPanics(t, func() {
		glogLogger.Infof("formatted with custom depth")
		glogLogger.Debug("structured on a zero value")
		glogLogger.Info("structured on a zero value", "key", "value")
		glogLogger.WarnContext(context.Background(), "structured with context")
		glogLogger.Error("structured error")
	}, "a GlogLogger built as a struct literal should not panic")
}

// Fatal paths.

// errBox lets a possibly-nil error be stored in an atomic.Value.
type errBox struct{ err error }

// emittedRecord is one structured record captured instead of being handed to glog.
type emittedRecord struct {
	level slog.Level
	line  string
}

// fatalRecorder captures what a fatal path would have done rather than letting
// it terminate the test process: the structured records that would have gone to
// glog, and Fatalf's delegation to glog.FatalDepthf.
type fatalRecorder struct {
	logger     *GlogLogger
	emitted    []emittedRecord
	glogFatalf []logCall
}

func newTestFatalLogger(t *testing.T) *fatalRecorder {
	t.Helper()
	// Every fatal path runs the process-global exit hooks, which run at most
	// once — each test needs its own registry.
	withCleanExitHooks(t)

	r := &fatalRecorder{}
	r.logger = newGlogLogger(1)
	r.logger.handler = &glogHandler{
		emit: func(_ context.Context, level slog.Level, _ int, line string) {
			r.emitted = append(r.emitted, emittedRecord{level: level, line: line})
		},
	}
	r.logger.fatalDepthf = func(_ int, format string, args ...any) {
		r.glogFatalf = append(r.glogFatalf, logCall{format, args})
	}
	return r
}

func TestGlogLogger_FatalEmitsAtFatalLevel(t *testing.T) {
	r := newTestFatalLogger(t)

	r.logger.Fatal("fatal error message", "key", "value")

	require.Len(t, r.emitted, 1)
	assert.Equal(t, LevelFatal, r.emitted[0].level,
		"Fatal must emit at LevelFatal so the handler routes it to glog.FatalDepth")
	assert.Equal(t, "fatal error message, key=value", r.emitted[0].line)
}

func TestGlogLogger_FatalContextEmitsAtFatalLevel(t *testing.T) {
	r := newTestFatalLogger(t)

	r.logger.FatalContext(context.Background(), "fatal with context", "key", "value")

	require.Len(t, r.emitted, 1)
	assert.Equal(t, LevelFatal, r.emitted[0].level)
	assert.Equal(t, "fatal with context, key=value", r.emitted[0].line)
}

// TestGlogLogger_FatalRunsExitHooksFirst pins the ordering both fatal paths
// share: glog writes the record and terminates in a single call, so cleanup has
// to happen before it, or not at all.
func TestGlogLogger_FatalRunsExitHooksFirst(t *testing.T) {
	cases := map[string]func(l *GlogLogger){
		"Fatal":        func(l *GlogLogger) { l.Fatal("boom") },
		"FatalContext": func(l *GlogLogger) { l.FatalContext(context.Background(), "boom") },
	}
	for name, fatal := range cases {
		t.Run(name, func(t *testing.T) {
			r := newTestFatalLogger(t)

			var hookRanFirst atomic.Bool
			RegisterExitHook("cleanup", func(context.Context) error {
				hookRanFirst.Store(len(r.emitted) == 0)
				return nil
			})

			fatal(r.logger)

			assert.True(t, hookRanFirst.Load(), "exit hooks must run before glog terminates the process")
			assert.Len(t, r.emitted, 1, "the fatal record should still be emitted")
		})
	}
}

func TestGlogLogger_FatalfRunsExitHooksFirst(t *testing.T) {
	r := newTestFatalLogger(t)

	var hookRanFirst atomic.Bool
	RegisterExitHook("cleanup", func(context.Context) error {
		hookRanFirst.Store(len(r.glogFatalf) == 0)
		return nil
	})

	r.logger.Fatalf("fatal: %s", "reason")

	assert.True(t, hookRanFirst.Load(), "exit hooks must run before glog terminates the process")
	require.Len(t, r.glogFatalf, 1, "Fatalf should still delegate to glog's fatal path")
	assert.Equal(t, "fatal: %s", r.glogFatalf[0].msg)
	assert.Equal(t, []any{"reason"}, r.glogFatalf[0].args)
}

// TestGlogLogger_FatalContextGivesHooksAnUncancelledContext covers the common
// case of a fatal raised while handling a request whose context is already
// cancelled: the hooks still need a usable budget to flush, but should keep the
// request's values.
func TestGlogLogger_FatalContextGivesHooksAnUncancelledContext(t *testing.T) {
	r := newTestFatalLogger(t)

	var hookErr atomic.Value  // error, wrapped so a nil error is still storable
	var sawValue atomic.Value // any
	RegisterExitHook("flush", func(ctx context.Context) error {
		hookErr.Store(errBox{ctx.Err()})
		if v := ctx.Value(testRequestIDKey); v != nil {
			sawValue.Store(v)
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), testRequestIDKey, "req-1"))
	cancel()

	r.logger.FatalContext(ctx, "fatal during a cancelled request")

	box, ok := hookErr.Load().(errBox)
	require.True(t, ok, "the hook should have run")
	assert.NoError(t, box.err, "hooks should not inherit the caller's cancellation")
	assert.Equal(t, "req-1", sawValue.Load(), "hooks should still see the caller's context values")
}

// fatalSubprocessEnv selects which fatal path the re-executed test binary should
// take. See TestGlogLogger_FatalTerminatesLikeGlog.
const fatalSubprocessEnv = "PBS_LOGGER_FATAL_SUBPROCESS"

// TestGlogLogger_FatalTerminatesLikeGlog is the only test that exercises real
// termination, by re-executing this test binary. It is what backs the Exiter
// contract: both fatal paths must run the exit hooks, write an F-severity record
// through glog, dump every goroutine's stack, and exit 2. A seam-based test
// cannot check any of that, because all of it happens inside glog.
func TestGlogLogger_FatalTerminatesLikeGlog(t *testing.T) {
	if mode := os.Getenv(fatalSubprocessEnv); mode != "" {
		fatalSubprocessChild(mode)
		return
	}

	for _, mode := range []string{"formatted", "structured"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestGlogLogger_FatalTerminatesLikeGlog")
			cmd.Env = append(os.Environ(), fatalSubprocessEnv+"="+mode)
			out, err := cmd.CombinedOutput()

			var exitErr *exec.ExitError
			require.ErrorAsf(t, err, &exitErr, "the child should have terminated; output:\n%s", out)
			assert.Equal(t, 2, exitErr.ExitCode(), "fatal should exit 2, matching glog")

			output := string(out)
			assert.Contains(t, output, "exit hook ran", "the exit hook should run before termination")
			assert.Contains(t, output, "child is dying", "the fatal record should be written")
			assert.Regexp(t, `(?m)^F\d{4} \d{2}:\d{2}:\d{2}\.\d{6}\s+\d+ glog_test\.go:\d+\] child is dying`,
				output, "the fatal record should carry glog's format and name the call site")
			assert.Contains(t, output, "goroutine ", "glog should dump all goroutine stacks")
		})
	}
}

// fatalSubprocessChild is the body run by the re-executed binary. It must not
// use the testing API: it is expected to die.
func fatalSubprocessChild(mode string) {
	_ = flag.Set("logtostderr", "true")

	RegisterExitHook("probe", func(context.Context) error {
		fmt.Fprintln(os.Stderr, "exit hook ran")
		return nil
	})

	log := NewGlogLogger()
	switch mode {
	case "formatted":
		log.Fatalf("child is dying: %s", "formatted")
	case "structured":
		log.Fatal("child is dying", "path", "structured")
	}
}
