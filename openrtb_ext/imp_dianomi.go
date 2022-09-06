package openrtb_ext

import (
	"encoding/json"
)

type ExtImpDianomi struct {
	SmartadId json.Number `json:"smartadId,omitempty"`
	PriceType string      `json:"priceType,omitempty"`
}
