package pubstack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/go-units"
	"github.com/prebid/prebid-server/v3/logger"
)

func fetchConfig(client *http.Client, endpoint *url.URL) (*Configuration, error) {
	res, err := client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		// read the entire response body to ensure full connection reuse if there's an
		// error while decoding the json
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			logger.Error("[pubstack] Draining config response body failed: %v", err)
		}
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
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
