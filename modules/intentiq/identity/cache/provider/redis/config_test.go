package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{Host: "localhost", Port: 6379}
	assert.NoError(t, valid.Validate())
	assert.NoError(t, Config{Host: "10.0.0.5", Port: 1}.Validate(), "an IP is a valid host")
	assert.NoError(t, Config{Host: "localhost", Port: 6379, ConnectTimeoutMs: 2_000}.Validate(),
		"an explicit connect timeout is valid")

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"no host", Config{Port: 6379}, "host"},
		{"malformed host", Config{Host: "not a host", Port: 6379}, "host"},
		{"no port", Config{Host: "localhost"}, "port"},
		{"port too high", Config{Host: "localhost", Port: 70000}, "port"},
		{"negative port", Config{Host: "localhost", Port: -1}, "port"},
		{"negative connect timeout", Config{Host: "localhost", Port: 6379, ConnectTimeoutMs: -1}, "connect_timeout_ms"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorContains(t, tc.cfg.Validate(), tc.want)
		})
	}
}

func TestConfigConnectTimeoutDefaultsWhenUnset(t *testing.T) {
	tests := []struct {
		name string
		ms   int
		want time.Duration
	}{
		{"unset falls back to the default", 0, 5 * time.Second},
		{"negative falls back to the default", -1, 5 * time.Second},
		{"explicit value is honoured", 250, 250 * time.Millisecond},
		{"explicit value above the default is honoured", 30_000, 30 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Config{ConnectTimeoutMs: tc.ms}.ConnectTimeout())
		})
	}
}
