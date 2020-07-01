package pubstack

import (
	"encoding/json"
	"github.com/docker/go-units"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics/clients"
	"net/url"
	"os"
	"path"
	"time"
)

func (p *PubstackModule) fetchAndUpdateConfig(refreshDelay time.Duration, endCh chan os.Signal) {
	tick := time.NewTicker(refreshDelay)

	for {
		select {
		case <-tick.C:
			config, err := fetchConfig(p.cfg.ScopeId, p.cfg.Endpoint)
			if err != nil {
				glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
				continue
			}
			p.configCh <- config
		case <-endCh:
			return
		}
	}
}

func fetchConfig(scope string, intake string) (*Configuration, error) {
	u, err := url.Parse(intake)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "bootstrap")
	q, _ := url.ParseQuery(u.RawQuery)

	q.Add("scopeId", scope)
	u.RawQuery = q.Encode()

	res, err := clients.GetDefaultHttpInstance().Get(u.String())
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
