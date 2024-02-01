package pubstack

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchConfig(t *testing.T) {
	configResponse := `{
		"scopeId":  "scopeId",
		"endpoint": "https://pubstack.io",
		"features": {
			"auction":    true,
			"cookiesync": true,
			"amp":        true,
			"setuid":     false,
			"video":      false
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		res.Write([]byte(configResponse))
	}))

	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg, _ := fetchConfig(server.Client(), endpoint)

	assert.Equal(t, "scopeId", cfg.ScopeID)
	assert.Equal(t, "https://pubstack.io", cfg.Endpoint)
	assert.Equal(t, true, cfg.Features[auction])
	assert.Equal(t, true, cfg.Features[cookieSync])
	assert.Equal(t, true, cfg.Features[amp])
	assert.Equal(t, false, cfg.Features[setUID])
	assert.Equal(t, false, cfg.Features[video])
}

func TestFetchConfig_Error(t *testing.T) {
	configResponse := `{
		"error":  "scopeId",
	}`

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		res.Write([]byte(configResponse))
	}))

	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg, err := fetchConfig(server.Client(), endpoint)

	assert.Nil(t, cfg)
	assert.Error(t, err)
}

func TestIsSameAs(t *testing.T) {
	copyConfig := func(conf Configuration) *Configuration {
		newConfig := conf
		newConfig.Features = make(map[string]bool)
		for k := range conf.Features {
			newConfig.Features[k] = conf.Features[k]
		}
		return &newConfig
	}

	a := &Configuration{
		ScopeID:  "scopeId",
		Endpoint: "endpoint",
		Features: map[string]bool{
			"auction":    true,
			"cookiesync": true,
			"amp":        true,
			"setuid":     false,
			"video":      false,
		},
	}

	assert.True(t, a.isSameAs(copyConfig(*a)))

	b := copyConfig(*a)
	b.ScopeID = "anotherId"
	assert.False(t, a.isSameAs(b))

	b = copyConfig(*a)
	b.Endpoint = "anotherEndpoint"
	assert.False(t, a.isSameAs(b))

	b = copyConfig(*a)
	b.Features["auction"] = true
	assert.True(t, a.isSameAs(b))
	b.Features["auction"] = false
	assert.False(t, a.isSameAs(b))
}

func TestClone(t *testing.T) {
	config := &Configuration{
		ScopeID:  "scopeId",
		Endpoint: "endpoint",
		Features: map[string]bool{
			"auction":    true,
			"cookiesync": true,
			"amp":        true,
			"setuid":     false,
			"video":      false,
		},
	}

	clone := config.clone()

	assert.Equal(t, config, clone)
	assert.NotSame(t, config, clone)
}

func TestDisableAllFeatures(t *testing.T) {
	config := &Configuration{
		ScopeID:  "scopeId",
		Endpoint: "endpoint",
		Features: map[string]bool{
			"auction":    true,
			"cookiesync": true,
			"amp":        true,
			"setuid":     false,
			"video":      false,
		},
	}

	expected := &Configuration{
		ScopeID:  "scopeId",
		Endpoint: "endpoint",
		Features: map[string]bool{
			"auction":    false,
			"cookiesync": false,
			"amp":        false,
			"setuid":     false,
			"video":      false,
		},
	}

	disabled := config.disableAllFeatures()

	assert.Equal(t, expected, disabled)
	assert.Same(t, config, disabled)
}
