package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gathered returns the metric family names currently held by reg.
func gathered(t *testing.T, reg *prometheus.Registry) []string {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}
	return names
}

// The host's registry is the only way out for these series, so assert they actually land in it.
func TestNewRegistersIntoHostRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()

	m := New(reg, true)
	assert.IsType(t, &promMetrics{}, m, "enabled with a registerer -> prom")

	m.Enriched("p1") // give the gather something to report
	assert.Contains(t, gathered(t, reg), Prefix+"enriched_total")
}

func TestNewDisabledYieldsNoop(t *testing.T) {
	reg := prometheus.NewRegistry()

	m := New(reg, false)
	assert.IsType(t, Noop{}, m, "disabled -> noop")

	m.Enriched("p1")
	assert.Empty(t, gathered(t, reg), "a disabled module must not register anything")
}

// A host with no Prometheus registry has nowhere to publish to; recording degrades instead of
// failing startup.
func TestNewNilRegistererYieldsNoop(t *testing.T) {
	m := New(nil, true)
	assert.IsType(t, Noop{}, m, "no registerer -> noop")
	assert.NotPanics(t, func() { m.Enriched("p1") }, "recording must stay safe with no registerer")
}

// Builder registers on construction, so a second module sharing one registry is a programming error
// that must surface loudly rather than dropping series.
func TestNewTwiceIntoOneRegistryPanics(t *testing.T) {
	reg := prometheus.NewRegistry()
	New(reg, true)

	assert.Panics(t, func() { New(reg, true) })
}
