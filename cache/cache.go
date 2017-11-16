package cache

import (
	"github.com/prebid/prebid-server/pbs/buckets"
	"encoding/json"
)

type Domain struct {
	Domain string `json:"domain"`
}

type App struct {
	Bundle string `json:"bundle"`
}

type Account struct {
	ID               string `json:"id"`
	PriceGranularity buckets.PriceGranularity `json:"price_granularity"`
}

type Configuration struct {
	Type string `json:"type"` // required
}

type Cache interface {
	Close() error
	Accounts() AccountsService
	Config() ConfigService
}

type AccountsService interface {
	Get(string) (*Account, error)
	Set(*Account) error
}

type ConfigService interface {
	Get(string) (string, error)
	Set(string, string) error
}

// ConfigFetcher knows how to fetch OpenRTB configs by id.
// Implementations must be safe for concurrent access by multiple goroutines.
//
// A config is basically a "partial" OpenRTB request.
// The Endpoint merges these into the HTTP Request JSON before unmarhsalling it
// into the OpenRTB Request which gets sent into the Exchange.
type ConfigFetcher interface {
	// GetConfigs fetches configs for the given IDs.
	// The returned map will have keys for every ID, unless errors exist.
	GetConfigs(ids []string) (map[string]json.RawMessage, []error)
}
