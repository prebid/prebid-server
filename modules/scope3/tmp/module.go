// Package tmp implements a Prebid Server module for AdCP Trusted Match Protocol.
package tmp

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Builder is the entry point for the module.
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &Module{cfg: cfg}, nil
}

// Config holds module configuration.
type Config struct{}

// Module implements the Scope3 TMP module.
type Module struct {
	cfg Config
}
