package pubstack

import (
	"encoding/json"
	"github.com/docker/go-units"
	"github.com/golang/glog"
	"net/http"
	"net/url"
	"path"
	"time"
)

func (p *PubstackModule) fetchAndUpdateConfig(refreshDelay time.Duration) {
	tick := time.NewTicker(refreshDelay)

	for {
		select {
		case <-tick.C:
			config, err := fetchConfig(p.httpClient, p.cfg.ScopeId, p.cfg.Endpoint)
			if err != nil {
				glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
				continue
			}
			p.configCh <- config
		case <-p.endCh:
			return
		}
	}
}

func fetchConfig(client *http.Client, scope string, intake string) (*Configuration, error) {
	u, err := url.Parse(intake)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "bootstrap")
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	q.Add("scopeId", scope)
	u.RawQuery = q.Encode()

	res, err := client.Get(u.String())
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
