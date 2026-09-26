package vastlint

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type config struct {
	Enabled       bool `json:"enabled"`
	RejectRevenue bool `json:"reject_revenue"`
}

func parseConfig(raw json.RawMessage) (config, error) {
	if len(raw) == 0 {
		return config{}, nil
	}
	var cfg config
	if err := jsonutil.UnmarshalValid(raw, &cfg); err != nil {
		return config{}, fmt.Errorf("failed to parse openadtech.vastlint config: %s", err)
	}
	return cfg, nil
}
