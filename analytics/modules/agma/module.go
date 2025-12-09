package agma

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func Builder(cfg json.RawMessage, deps analyticsdeps.Deps) (analytics.Module, error) {
	if deps.HTTPClient == nil || deps.Clock == nil {
		return nil, nil
	}

	var c Config
	if len(cfg) > 0 {
		if err := jsonutil.Unmarshal(cfg, &c); err != nil {
			return nil, err
		}
	}

	if !c.Enabled || c.Endpoint.Url == "" {
		return nil, nil
	}

	return NewModule(deps.HTTPClient, c, deps.Clock)
}
