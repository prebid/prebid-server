package openrtb_ext

import (
	"encoding/json"
)

type ExtImpAdhese struct {
	Account  string          `json:"account"`
	Location string          `json:"location"`
	Format   string          `json:"format"`
	Keywords json.RawMessage `json:"targets,omitempty"`
}
