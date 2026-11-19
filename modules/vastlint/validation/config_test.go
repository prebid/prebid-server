package validation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigValidatesNewFields(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{`{"mode":"sync"}`, "mode must be"},
		{`{"cache_size":-1}`, "cache_size"},
		{`{"cache_ttl_seconds":-5}`, "cache_ttl_seconds"},
		{`{"async_queue":0}`, ""},
		{`{"async_queue":2000000}`, "async_queue must"},
		{`{"async_queue_bytes":10}`, "async_queue_bytes"},
	} {
		_, err := parseConfig(json.RawMessage(tc.raw))
		if tc.want == "" {
			assert.NoError(t, err, tc.raw)
		} else {
			require.Error(t, err, tc.raw)
			assert.Contains(t, err.Error(), tc.want)
		}
	}
	cfg, err := parseConfig(json.RawMessage(`{"enabled":true}`))
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.CacheSize)
	assert.Equal(t, 600, cfg.CacheTTLSeconds)
	assert.Equal(t, 4096, cfg.AsyncQueue)
	assert.Equal(t, 64<<20, cfg.AsyncQueueBytes)
	assert.True(t, *cfg.AnnotateRules)
}

func TestParseConfigDefaultsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{`{"report_interval_seconds":0}`, ""},
		{`{"report_interval_seconds":-1}`, "report_interval_seconds"},
		{`{"report_interval_seconds":4000}`, "report_interval_seconds"},
		{`{"pool_size":-1}`, "pool_size"},
		{`{"pool_size":4096}`, "pool_size"},
		{`{"max_document_bytes":-1}`, "max_document_bytes"},
		{`{"sample_percent":101}`, "sample_percent"},
		{`{"rules":{"VAST-2.0-inline-impression":"loud"}}`, "must be error, warning, info or off"},
		{`{"mode":`, "invalid module config"},
		{``, ""},
	} {
		_, err := parseConfig(json.RawMessage(tc.raw))
		if tc.want == "" {
			assert.NoError(t, err, tc.raw)
		} else {
			require.Error(t, err, tc.raw)
			assert.Contains(t, err.Error(), tc.want)
		}
	}
	cfg, err := parseConfig(json.RawMessage(`{"reject_on":["b","a"]}`))
	require.NoError(t, err)
	assert.Equal(t, 0, *cfg.ReportIntervalSeconds, "periodic log is opt-in")
	assert.Equal(t, 256<<10, cfg.MaxDocumentBytes)
	assert.Equal(t, 100, cfg.SamplePercent)
	assert.Equal(t, []string{"a", "b"}, cfg.RejectOn, "reject_on is sorted for lookup")
	assert.True(t, cfg.rejects("a"))
	assert.False(t, cfg.rejects("c"))
}

func TestAccountRulesMergeOverHostRules(t *testing.T) {
	host := cfgWith(modeMonitor)
	host.Rules = ruleLevels{"A": "off", "B": "warning"}
	known := newCatalog([]string{"A", "B", "C"})
	eff, err := host.effective(json.RawMessage(`{"rules":{"B":"error","C":"off"}}`), known)
	require.NoError(t, err)
	assert.Equal(t, ruleLevels{"A": "off", "B": "error", "C": "off"}, eff.Rules)
	assert.Equal(t, ruleLevels{"A": "off", "B": "warning"}, host.Rules, "host config is not modified")
	eff, err = host.effective(json.RawMessage(`{"sample_percent":50}`), known)
	require.NoError(t, err)
	assert.Equal(t, host.Rules, eff.Rules)
	assert.Equal(t, 50, eff.SamplePercent)
}

func TestRulesAcceptLoaderFriendlyForms(t *testing.T) {
	cat := newCatalog([]string{"VAST-2.0-adsystem-no-version", "VAST-2.0-inline-impression"})
	for _, tc := range []struct {
		name, raw string
	}{
		{"object", `{"rules":{"VAST-2.0-adsystem-no-version":"off"}}`},
		{"object lowercased and split on dots by a YAML loader", `{"rules":{"vast-2":{"0-adsystem-no-version":"off"}}}`},
		{"list of id=level", `{"rules":["VAST-2.0-adsystem-no-version=off"]}`},
		{"list of id:level", `{"rules":["vast-2.0-adsystem-no-version: off"]}`},
		{"list of objects", `{"rules":[{"id":"VAST-2.0-adsystem-no-version","level":"off"}]}`},
	} {
		cfg, err := parseConfig(json.RawMessage(tc.raw))
		require.NoError(t, err, tc.name)
		rules, _, err := cat.resolve(cfg.Rules, nil)
		require.NoError(t, err, tc.name)
		assert.Equal(t, ruleLevels{"VAST-2.0-adsystem-no-version": "off"}, rules, tc.name)
	}
	for _, raw := range []string{`{"rules":["no-equals"]}`, `{"rules":[{"id":"x"}]}`, `{"rules":[7]}`, `{"rules":"off"}`, `{"rules":{"a":{"b":7}}}`} {
		_, err := parseConfig(json.RawMessage(raw))
		assert.Error(t, err, raw)
	}
	_, rejectOn, err := cat.resolve(nil, []string{"vast-2.0-inline-impression", "VAST-2.0-ADSYSTEM-NO-VERSION"})
	require.NoError(t, err)
	assert.Equal(t, []string{"VAST-2.0-adsystem-no-version", "VAST-2.0-inline-impression"}, rejectOn, "reject_on ids are canonicalized and sorted")
	_, _, err = cat.resolve(ruleLevels{"VAST-2.0-nope": "off"}, nil)
	assert.Error(t, err)
}
