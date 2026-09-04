package aerospike

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validConfig() Config {
	return Config{Host: "localhost", Port: 3000, Namespace: "prebid", Set: "identity"}
}

func TestConfigValidate(t *testing.T) {
	assert.NoError(t, validConfig().Validate())

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"no host", func(c *Config) { c.Host = "" }, "host"},
		{"malformed host", func(c *Config) { c.Host = "not a host" }, "host"},
		{"no port", func(c *Config) { c.Port = 0 }, "port"},
		{"port too high", func(c *Config) { c.Port = 70000 }, "port"},
		{"no namespace", func(c *Config) { c.Namespace = "" }, "namespace"},
		{"no set", func(c *Config) { c.Set = "" }, "set"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(&c)
			assert.ErrorContains(t, c.Validate(), tc.want)
		})
	}
}

func TestClientPolicyZeroIsValid(t *testing.T) {
	assert.NoError(t, ClientPolicy{}.Validate())
	assert.NoError(t, validConfig().Validate(), "config with no client_policy")
}

func TestClientPolicyRejectsNegatives(t *testing.T) {
	tests := []struct {
		name   string
		policy ClientPolicy
	}{
		{"queue size", ClientPolicy{ConnectionQueueSize: -1}},
		{"min connections", ClientPolicy{MinConnectionsPerNode: -1}},
		{"connect timeout", ClientPolicy{ConnectTimeoutMs: -1}},
		{"idle timeout", ClientPolicy{IdleTimeoutMs: -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, tc.policy.Validate())

			c := validConfig()
			c.Policy = tc.policy
			assert.Error(t, c.Validate(), "a bad policy must fail the enclosing config too")
		})
	}
}

func TestPortRuleAcceptsRealPorts(t *testing.T) {
	for _, port := range []int{1, 3000, 65535} {
		c := validConfig()
		c.Port = port
		assert.NoError(t, c.Validate(), "port %d must be accepted", port)
	}
}
