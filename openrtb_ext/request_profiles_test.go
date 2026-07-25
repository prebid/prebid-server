package openrtb_ext

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- TestExtRequestPrebid_ProfilesSerialization ---

func TestExtRequestPrebid_ProfilesSerialization(t *testing.T) {
	t.Run("profiles field present when non-empty", func(t *testing.T) {
		ep := ExtRequestPrebid{Profiles: []string{"a", "b"}}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"profiles":["a","b"]`)
	})

	t.Run("profiles field absent when empty struct", func(t *testing.T) {
		ep := ExtRequestPrebid{}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.False(t, strings.Contains(string(data), `"profiles"`), "profiles key should be absent")
	})
}

// --- TestExtRequestPrebid_OutputFormat ---

func TestExtRequestPrebid_OutputFormat(t *testing.T) {
	t.Run("of field set", func(t *testing.T) {
		ep := ExtRequestPrebid{OutputFormat: "vast4"}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"of":"vast4"`)
	})

	t.Run("om field set", func(t *testing.T) {
		ep := ExtRequestPrebid{OutputModule: "prebid.ctv"}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"om":"prebid.ctv"`)
	})
}

// --- TestExtRequestPrebidServer_HTTPMethod ---

// The JSON name must stay aligned with prebid-server-java's `http_method`
// (ExtRequestPrebidServer.httpMethod under Jackson SNAKE_CASE).
func TestExtRequestPrebidServer_HTTPMethod(t *testing.T) {
	t.Run("http_method present when set", func(t *testing.T) {
		s := ExtRequestPrebidServer{HTTPMethod: "GET"}
		data, err := json.Marshal(s)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"http_method":"GET"`)
	})

	t.Run("http_method absent when empty", func(t *testing.T) {
		s := ExtRequestPrebidServer{}
		data, err := json.Marshal(s)
		require.NoError(t, err)
		assert.False(t, strings.Contains(string(data), `"http_method"`), "http_method should be absent")
	})

	t.Run("http_method round-trips from JSON", func(t *testing.T) {
		var s ExtRequestPrebidServer
		require.NoError(t, json.Unmarshal([]byte(`{"http_method":"POST"}`), &s))
		assert.Equal(t, "POST", s.HTTPMethod)
	})
}

// --- TestExtImpPrebid_Profiles ---

func TestExtImpPrebid_Profiles(t *testing.T) {
	t.Run("profiles field present when non-empty", func(t *testing.T) {
		ep := ExtImpPrebid{Profiles: []string{"highbandwidth"}}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"profiles":["highbandwidth"]`)
	})

	t.Run("profiles field absent when empty struct", func(t *testing.T) {
		ep := ExtImpPrebid{}
		data, err := json.Marshal(ep)
		require.NoError(t, err)
		assert.False(t, strings.Contains(string(data), `"profiles"`), "profiles key should be absent")
	})
}
