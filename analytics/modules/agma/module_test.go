package agma

import (
	"encoding/json"
	"net/http"
	"testing"

	bjclock "github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/stretchr/testify/assert"
)

func newDepsWith(client *http.Client, c bjclock.Clock) analyticsdeps.Deps {
	return analyticsdeps.Deps{
		HTTPClient: client,
		Clock:      c,
	}
}

func TestBuilderNilDeps(t *testing.T) {
	mod, err := Builder(nil, analyticsdeps.Deps{})
	assert.NoError(t, err)
	assert.Nil(t, mod)

	client := &http.Client{}
	mod, err = Builder(nil, analyticsdeps.Deps{HTTPClient: client})
	assert.NoError(t, err)
	assert.Nil(t, mod)

	clk := bjclock.NewMock()
	mod, err = Builder(nil, analyticsdeps.Deps{Clock: clk})
	assert.NoError(t, err)
	assert.Nil(t, mod)
}

func TestBuilderEmptyConfig(t *testing.T) {
	client := &http.Client{}
	clk := bjclock.NewMock()
	deps := newDepsWith(client, clk)

	mod, err := Builder([]byte(`{}`), deps)
	assert.NoError(t, err)
	assert.Nil(t, mod)
}

func TestBuilderDisabled(t *testing.T) {
	client := &http.Client{}
	clk := bjclock.NewMock()
	deps := newDepsWith(client, clk)

	cfg := Config{
		Enabled: false,
		Endpoint: EndpointConfig{
			Url: "https://agma.example.com/collect",
		},
	}
	raw, err := json.Marshal(cfg)
	assert.NoError(t, err)

	mod, err := Builder(raw, deps)
	assert.NoError(t, err)
	assert.Nil(t, mod)
}

func TestBuilderNoEndpointURL(t *testing.T) {
	client := &http.Client{}
	clk := bjclock.NewMock()
	deps := newDepsWith(client, clk)

	cfg := Config{
		Enabled: true,
		Endpoint: EndpointConfig{
			Url: "",
		},
	}
	raw, err := json.Marshal(cfg)
	assert.NoError(t, err)

	mod, err := Builder(raw, deps)
	assert.NoError(t, err)
	assert.Nil(t, mod)
}

func TestBuilderInvalidConfig(t *testing.T) {
	client := &http.Client{}
	clk := bjclock.NewMock()
	deps := newDepsWith(client, clk)

	mod, err := Builder([]byte(`{`), deps)
	assert.Error(t, err)
	assert.Nil(t, mod)
}

func TestBuilderValidConfig(t *testing.T) {
	client := &http.Client{}
	clk := bjclock.NewMock()
	deps := newDepsWith(client, clk)

	cfg := Config{
		Enabled: true,
		Endpoint: EndpointConfig{
			Url:     "https://agma.example.com/collect",
			Timeout: "2s",
			Gzip:    true,
		},
		Buffers: BufferConfig{
			BufferSize: "1MB",
			EventCount: 100,
			Timeout:    "1s",
		},
		Accounts: []AccountConfig{
			{
				Code:        "acc-1",
				PublisherId: "pub-1",
				SiteAppId:   "site-1",
			},
		},
	}
	raw, err := json.Marshal(cfg)
	assert.NoError(t, err)

	mod, err := Builder(raw, deps)
	assert.NoError(t, err)
	assert.NotNil(t, mod)

	_, ok := mod.(analytics.Module)
	assert.True(t, ok)
}
