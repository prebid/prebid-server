package tracing

import (
	"sync"
	"time"
)

// Rule is a hardcoded tracing rule for a single partner.
type Rule struct {
	PartnerID          string
	Duration           time.Duration
	TracePacketsAmount int
}

type partnerState struct {
	firstSeen time.Time
	count     int
}

// Tracer decides, for a given partner, whether a new trace packet may still be collected,
// based on the hardcoded rules and previously observed activity for that partner.
type Tracer struct {
	mu    sync.Mutex
	rules map[string]Rule
	state map[string]*partnerState
}

// NewTracer builds a Tracer from a set of hardcoded rules, keyed by PartnerID.
func NewTracer(rules []Rule) *Tracer {
	byPartner := make(map[string]Rule, len(rules))
	for _, rule := range rules {
		byPartner[rule.PartnerID] = rule
	}
	return &Tracer{
		rules: byPartner,
		state: make(map[string]*partnerState),
	}
}

// Allow reports whether a trace packet for partnerID may be collected at time now.
//
// The first call for a partner starts its tracing window. Every allowed call consumes one
// unit of that partner's TracePacketsAmount budget. Once either the Duration since the first
// call or the TracePacketsAmount budget is exhausted, tracing stops permanently for that
// partner - it never resumes, even if called again later.
func (t *Tracer) Allow(partnerID string, now time.Time) bool {
	rule, ok := t.rules[partnerID]
	if !ok {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.state[partnerID]
	if !ok {
		state = &partnerState{firstSeen: now}
		t.state[partnerID] = state
	}

	if now.Sub(state.firstSeen) > rule.Duration {
		return false
	}
	if state.count >= rule.TracePacketsAmount {
		return false
	}

	state.count++
	return true
}
