package pubstack

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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
		res.WriteHeader(200)
	}))

	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg, _ := fetchConfig(server.Client(), endpoint)

	assert.Equal(t, cfg.ScopeID, "scopeId")
	assert.Equal(t, cfg.Endpoint, "https://pubstack.io")
	assert.Equal(t, cfg.Features[auction], true)
	assert.Equal(t, cfg.Features[cookieSync], true)
	assert.Equal(t, cfg.Features[amp], true)
	assert.Equal(t, cfg.Features[setUID], false)
	assert.Equal(t, cfg.Features[video], false)
}

func TestFetchConfig_Error(t *testing.T) {
	configResponse := `{
		"error":  "scopeId",
	}`

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		res.Write([]byte(configResponse))
		res.WriteHeader(200)
	}))

	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg, err := fetchConfig(server.Client(), endpoint)

	assert.Nil(t, cfg)
	assert.NotNil(t, err)
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
