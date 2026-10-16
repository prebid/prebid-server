package logger

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/golang/glog"
)

var _ slog.Handler = (*glogHandler)(nil)

// emitFunc writes one already-rendered log line. depth tells glog how many
// stack frames to skip when working out the file:line to attribute the record
// to. Swapping it is how tests observe the Fatal path without dying.
type emitFunc func(ctx context.Context, level slog.Level, depth int, line string)

// glogHandler is a slog.Handler that emits records through glog itself rather
// than writing them to an io.Writer of its own.
//
// Routing through glog is the point: structured records then get exactly what
// the printf-style methods get — glog's line format including file:line, its
// severity routing across the INFO/WARNING/ERROR/FATAL files, --log_dir,
// --logtostderr, --stderrthreshold, rotation, and the flush-dump-exit sequence
// on fatal. A handler that formatted glog-ish lines onto stderr would look
// similar and behave differently: invisible to --log_dir deployments, immune to
// every glog flag, and rejected by glog log parsers.
type glogHandler struct {
	// depth is the number of frames between the emit call and the caller whose
	// file:line should be reported. See structuredCallDepth.
	depth int

	// preformatted holds attributes bound by WithAttrs, already rendered.
	preformatted string
	groups       []string

	// emit is a test seam; nil means emitToGlog.
	emit emitFunc
}

// Enabled reports whether a record at level should be built.
//
// It is always true because glog, not this handler, decides what survives:
// severity thresholds, --log_dir and --logtostderr are applied downstream in
// glog's sinks. Note that Debug maps onto glog's INFO severity, which glog has
// no way to suppress — the same is true of the existing Debugf, and the two
// deliberately behave alike so that migrating a call site cannot silently
// change what gets logged.
func (h *glogHandler) Enabled(context.Context, slog.Level) bool { return true }

// Handle renders the record as "message, key=value, key=value" — glog supplies
// the severity letter, timestamp, thread id and file:line prefix — and hands it
// to glog at the matching severity.
func (h *glogHandler) Handle(ctx context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	b.WriteString(h.preformatted)
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(&b, h.groups, a)
		return true
	})

	emit := h.emit
	if emit == nil {
		emit = emitToGlog
	}
	emit(ctx, r.Level, h.depth, b.String())
	return nil
}

func (h *glogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	var b strings.Builder
	b.WriteString(h.preformatted)
	for _, a := range attrs {
		appendAttr(&b, h.groups, a)
	}
	clone := *h
	clone.preformatted = b.String()
	return &clone
}

func (h *glogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(slices.Clip(h.groups), name)
	return &clone
}

// emitToGlog is the production emitFunc. glog has no Debug severity, so Debug
// records go to INFO exactly as Debugf does; LevelFatal routes to glog's fatal
// path, which writes the record to every severity file, dumps the stacks of all
// goroutines, flushes and exits 2.
//
// The *ContextDepth variants are used so a glog sink that reads request context
// still sees it: dropping ctx here is what would make logger.InfoContext no
// better than logger.Info.
func emitToGlog(ctx context.Context, level slog.Level, depth int, line string) {
	switch {
	case level < slog.LevelWarn:
		glog.InfoContextDepth(ctx, depth, line)
	case level < slog.LevelError:
		glog.WarningContextDepth(ctx, depth, line)
	case level < LevelFatal:
		glog.ErrorContextDepth(ctx, depth, line)
	default:
		glog.FatalContextDepth(ctx, depth, line)
	}
}

// appendAttr renders one attribute as ", key=value", flattening groups into
// dotted keys. Values are resolved so slog.LogValuer implementations are
// honored, and wholly empty attributes are dropped as slog.Handler requires.
func appendAttr(b *strings.Builder, groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		// A group with an empty key is inlined, per the slog.Handler contract.
		if a.Key != "" {
			groups = append(slices.Clip(groups), a.Key)
		}
		for _, ga := range attrs {
			appendAttr(b, groups, ga)
		}
		return
	}

	b.WriteString(", ")
	for _, g := range groups {
		b.WriteString(g)
		b.WriteByte('.')
	}
	b.WriteString(a.Key)
	b.WriteByte('=')
	b.WriteString(a.Value.String())
}
