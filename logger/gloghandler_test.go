package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zeroTime keeps rendered records deterministic: glog stamps its own timestamp,
// so the one on the slog.Record is never read.
var zeroTime = time.Time{}

// renderRecord runs one record through the handler and returns the line it would
// have handed to glog, so attribute rendering can be asserted directly.
func renderRecord(h *glogHandler, level slog.Level, msg string, args ...any) string {
	var got string
	rendered := *h
	rendered.emit = func(_ context.Context, _ slog.Level, _ int, line string) { got = line }

	record := slog.NewRecord(zeroTime, level, msg, 0)
	record.Add(args...)
	_ = rendered.Handle(context.Background(), record)
	return got
}

// secret exercises the slog.LogValuer contract: a handler must resolve values
// before rendering, so redaction and lazy formatting work.
type secret string

func (s secret) LogValue() slog.Value { return slog.StringValue("REDACTED") }

func TestGlogHandler_RendersAttributes(t *testing.T) {
	h := &glogHandler{}

	cases := []struct {
		name string
		args []any
		want string
	}{
		{"no attributes", nil, "message"},
		{"key value pairs", []any{"key", "value", "count", 7}, "message, key=value, count=7"},
		{"slog.Attr arguments", []any{slog.String("k", "v")}, "message, k=v"},
		{"resolves LogValuer", []any{"token", secret("hunter2")}, "message, token=REDACTED"},
		{"dangling key", []any{"lonely"}, "message, !BADKEY=lonely"},
		{
			"group becomes a dotted key",
			[]any{slog.Group("http", slog.Int("status", 200), slog.String("method", "GET"))},
			"message, http.status=200, http.method=GET",
		},
		{
			"nested groups",
			[]any{slog.Group("a", slog.Group("b", slog.String("c", "d")))},
			"message, a.b.c=d",
		},
		{
			"group with an empty key is inlined",
			[]any{slog.Group("", slog.String("k", "v"))},
			"message, k=v",
		},
		{"empty group is dropped", []any{slog.Group("empty")}, "message"},
		{"empty attribute is dropped", []any{slog.Attr{}}, "message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, renderRecord(h, slog.LevelInfo, "message", tc.args...))
		})
	}
}

func TestGlogHandler_WithAttrs(t *testing.T) {
	base := &glogHandler{}

	bound, ok := base.WithAttrs([]slog.Attr{slog.String("service", "pbs")}).(*glogHandler)
	require.True(t, ok)

	assert.Equal(t, "message, service=pbs, key=value",
		renderRecord(bound, slog.LevelInfo, "message", "key", "value"))
	assert.Equal(t, "message", renderRecord(base, slog.LevelInfo, "message"),
		"WithAttrs must not mutate the handler it was called on")
	assert.Same(t, base, base.WithAttrs(nil), "binding no attributes should not allocate")
}

func TestGlogHandler_WithGroup(t *testing.T) {
	base := &glogHandler{}

	grouped, ok := base.WithGroup("req").(*glogHandler)
	require.True(t, ok)
	nested, ok := grouped.WithGroup("http").(*glogHandler)
	require.True(t, ok)

	assert.Equal(t, "message, req.id=1", renderRecord(grouped, slog.LevelInfo, "message", "id", 1))
	assert.Equal(t, "message, req.http.status=200", renderRecord(nested, slog.LevelInfo, "message", "status", 200))
	assert.Equal(t, "message, req.id=1", renderRecord(grouped, slog.LevelInfo, "message", "id", 1),
		"a nested group must not leak back into its parent")
	assert.Same(t, base, base.WithGroup(""), "an empty group name should be ignored")
}

// TestGlogHandler_WithAttrsUnderGroup pins that attributes bound while a group is
// open keep that group's prefix, which is where a naive implementation that
// re-renders preformatted attributes gets it wrong.
func TestGlogHandler_WithAttrsUnderGroup(t *testing.T) {
	h, ok := (&glogHandler{}).WithGroup("req").(*glogHandler)
	require.True(t, ok)
	bound, ok := h.WithAttrs([]slog.Attr{slog.String("id", "abc")}).(*glogHandler)
	require.True(t, ok)

	assert.Equal(t, "message, req.id=abc, req.status=200",
		renderRecord(bound, slog.LevelInfo, "message", "status", 200))
}

// TestGlogHandler_PassesContextToTheEmitter pins that the Context variants are
// worth having: dropping ctx on the way to glog would make InfoContext identical
// to Info for any glog sink that reads request context.
func TestGlogHandler_PassesContextToTheEmitter(t *testing.T) {
	type ctxKey struct{}
	want := context.WithValue(context.Background(), ctxKey{}, "trace-1")

	var got context.Context
	h := &glogHandler{emit: func(ctx context.Context, _ slog.Level, _ int, _ string) { got = ctx }}

	_ = h.Handle(want, slog.NewRecord(zeroTime, slog.LevelInfo, "message", 0))

	require.NotNil(t, got)
	assert.Equal(t, "trace-1", got.Value(ctxKey{}))
}

func TestGlogHandler_EnabledAtEveryLevel(t *testing.T) {
	h := &glogHandler{}
	// glog owns the filtering decision; the handler must not second-guess it.
	for _, level := range []slog.Level{slog.LevelDebug - 4, slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError, LevelFatal} {
		assert.Truef(t, h.Enabled(context.Background(), level), "level %v should be enabled", level)
	}
}

// TestGlogHandler_ZeroValueEmitsToGlog pins the nil-emit fallback, so a handler
// built as a struct literal still reaches glog rather than dropping records.
func TestGlogHandler_ZeroValueEmitsToGlog(t *testing.T) {
	h := &glogHandler{}

	out := captureGlogStderr(t, func() {
		_ = h.Handle(context.Background(), slog.NewRecord(zeroTime, slog.LevelWarn, "zero value", 0))
	})

	assert.Contains(t, out, "zero value")
	require.NotEmpty(t, out)
	assert.Equal(t, "W", out[:1])
}
