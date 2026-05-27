package tmp

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func TestBuilder_EmptyConfig(t *testing.T) {
	m, err := Builder(json.RawMessage(`{}`), moduledeps.ModuleDeps{HTTPClient: &http.Client{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "router_url is required")
	require.Nil(t, m)
}

func TestBuilder_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError string
	}{
		{
			name:      "missing router_url",
			config:    `{"seller_agent_url":"https://example.com"}`,
			wantError: "router_url is required",
		},
		{
			name:      "missing seller_agent_url",
			config:    `{"router_url":"https://tmp.interchange.io"}`,
			wantError: "seller_agent_url is required",
		},
		{
			name:      "too many preserve_eids",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"user":{"preserve_eids":["a","b","c","d"]}}}`,
			wantError: "preserve_eids exceeds spec limit of 3 entries",
		},
		{
			name:      "negative lat_long_precision",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":-1}}}`,
			wantError: "lat_long_precision cannot be negative",
		},
		{
			name:      "lat_long_precision over 4",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","masking":{"enabled":true,"geo":{"lat_long_precision":5}}}`,
			wantError: "lat_long_precision cannot exceed 4 decimal places for privacy protection",
		},
		{
			name:      "negative timeout_ms",
			config:    `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com","timeout_ms":-1}`,
			wantError: "timeout_ms must be positive",
		},
		{
			name:   "valid minimal config",
			config: `{"router_url":"https://tmp.interchange.io","seller_agent_url":"https://example.com"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := moduledeps.ModuleDeps{HTTPClient: &http.Client{}}
			m, err := Builder(json.RawMessage(tc.config), deps)
			if tc.wantError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantError)
				require.Nil(t, m)
			} else {
				require.NoError(t, err)
				require.NotNil(t, m)
			}
		})
	}
}
