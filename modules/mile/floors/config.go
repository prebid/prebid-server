package floors

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Config struct {
	GeoLookupEndpoint string `json:"geo_lookup_endpoint"`
	GeoCacheTTLMS     int    `json:"geo_cache_ttl_ms"`
}

func (c *Config) GeoEnabled() bool {
	return c.GeoLookupEndpoint != ""
}

func (c *Config) GetGeoCacheTTL() time.Duration {
	return time.Duration(c.GeoCacheTTLMS) * time.Millisecond
}

func parseConfig(rawConfig json.RawMessage) (*Config, error) {
	cfg := &Config{
		GeoCacheTTLMS: 3000000,
	}

	if err := jsonutil.Unmarshal(rawConfig, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
