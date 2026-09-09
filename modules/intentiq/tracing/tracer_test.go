package tracing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTracerAllow(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		description string
		rules       []Rule
		partnerID   string
		calls       []time.Time // one Allow() call per entry, in order
		expected    []bool      // expected Allow() result per call
	}{
		{
			description: "unknown partner is never allowed",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: 5}},
			partnerID:   "unknown",
			calls:       []time.Time{baseTime},
			expected:    []bool{false},
		},
		{
			description: "first call for a known partner is allowed and starts the window",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: 5}},
			partnerID:   "p1",
			calls:       []time.Time{baseTime},
			expected:    []bool{true},
		},
		{
			description: "amount limit stops tracing once reached",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: 2}},
			partnerID:   "p1",
			calls:       []time.Time{baseTime, baseTime, baseTime},
			expected:    []bool{true, true, false},
		},
		{
			description: "zero amount never allows any packet",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: 0}},
			partnerID:   "p1",
			calls:       []time.Time{baseTime},
			expected:    []bool{false},
		},
		{
			description: "time limit stops tracing once duration since first trace elapses",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Minute, TracePacketsAmount: 100}},
			partnerID:   "p1",
			calls: []time.Time{
				baseTime,
				baseTime.Add(30 * time.Second),
				baseTime.Add(61 * time.Second),
			},
			expected: []bool{true, true, false},
		},
		{
			description: "call exactly at the duration boundary is still allowed",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Minute, TracePacketsAmount: 100}},
			partnerID:   "p1",
			calls: []time.Time{
				baseTime,
				baseTime.Add(time.Minute),
			},
			expected: []bool{true, true},
		},
		{
			description: "once stopped, tracing does not resume even if called again later",
			rules:       []Rule{{PartnerID: "p1", Duration: time.Minute, TracePacketsAmount: 1}},
			partnerID:   "p1",
			calls: []time.Time{
				baseTime,
				baseTime.Add(2 * time.Minute),
				baseTime.Add(3 * time.Minute),
			},
			expected: []bool{true, false, false},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Arrange
			tracer := NewTracer(test.rules)

			// Act
			actual := make([]bool, len(test.calls))
			for i, callTime := range test.calls {
				actual[i] = tracer.Allow(test.partnerID, callTime)
			}

			// Assert
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTracerAllowIsIndependentPerPartner(t *testing.T) {
	// Arrange
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tracer := NewTracer([]Rule{
		{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: 1},
		{PartnerID: "p2", Duration: time.Hour, TracePacketsAmount: 1},
	})

	// Act
	p1First := tracer.Allow("p1", baseTime)
	p2First := tracer.Allow("p2", baseTime)
	p1Second := tracer.Allow("p1", baseTime)
	p2Second := tracer.Allow("p2", baseTime)

	// Assert
	assert.True(t, p1First)
	assert.True(t, p2First)
	assert.False(t, p1Second)
	assert.False(t, p2Second)
}

func TestTracerAllowConcurrentCallsRespectAmountLimit(t *testing.T) {
	// Arrange
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const amount = 50
	tracer := NewTracer([]Rule{{PartnerID: "p1", Duration: time.Hour, TracePacketsAmount: amount}})

	const workers = 200
	results := make(chan bool, workers)

	// Act
	for i := 0; i < workers; i++ {
		go func() {
			results <- tracer.Allow("p1", now)
		}()
	}

	allowedCount := 0
	for i := 0; i < workers; i++ {
		if <-results {
			allowedCount++
		}
	}

	// Assert
	assert.Equal(t, amount, allowedCount)
}
