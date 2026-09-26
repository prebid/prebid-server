package vastlint

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Module validates VAST video adm in-process and counts revenue-impact findings.
type Module struct {
	cfg      config
	validate func(xml string) ([]finding, error)
	record   recorder
}

// Builder is the Prebid Server module entry point.
// The module stays off unless hooks.modules.openadtech.vastlint.enabled is true.
func Builder(raw json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	validate, err := newValidator()
	if err != nil {
		return nil, err
	}
	return Module{cfg: cfg, validate: validate}, nil
}
