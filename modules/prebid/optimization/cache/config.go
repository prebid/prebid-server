package cache

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

func newConfig(data json.RawMessage) (config, error) {
	var cfg config
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func validateConfig(cfg config) error {
	return nil
}

type config struct {
}
