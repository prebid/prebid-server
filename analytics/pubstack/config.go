package pubstack

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/go-units"
)

func fetchConfig(client *http.Client, endpoint *url.URL) (*Configuration, error) {
	res, err := client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func newBufferConfig(count int, size, duration string) (*bufferConfig, error) {
	pDuration, err := time.ParseDuration(duration)
	if err != nil {
		return nil, err
	}
	pSize, err := units.FromHumanSize(size)
	if err != nil {
		return nil, err
	}
	return &bufferConfig{
		pDuration,
		int64(count),
		pSize,
	}, nil
}

func (a *Configuration) isSameAs(b *Configuration) bool {
	sameEndpoint := a.Endpoint == b.Endpoint
	sameScopeID := a.ScopeID == b.ScopeID
	sameFeature := len(a.Features) == len(b.Features)
	for key := range a.Features {
		sameFeature = sameFeature && a.Features[key] == b.Features[key]
	}
	return sameFeature && sameEndpoint && sameScopeID
}

func (a *Configuration) clone() *Configuration {
	c := &Configuration{
		ScopeID:  a.ScopeID,
		Endpoint: a.Endpoint,
		Features: make(map[string]bool, len(a.Features)),
	}

	for k, v := range a.Features {
		c.Features[k] = v
	}

	return c
}

func (a *Configuration) disableAllFeatures() *Configuration {
	for k := range a.Features {
		a.Features[k] = false
	}
	return a
}
