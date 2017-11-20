package openrtb2_config

import (
	"encoding/json"
)

// ConfigFetcher knows how to fetch OpenRTB configs by id.
//
// Implementations must be safe for concurrent access by multiple goroutines, and callers are expected to
// create share instances as much as possible.
type ConfigFetcher interface {
	// GetConfigs fetches configs for the given IDs.
	// The returned map will have keys for every ID, unless errors exist.
	//
	// The returned objects should only be read from--never written to.
	GetConfigs(ids []string) (map[string]json.RawMessage, []error)
}
