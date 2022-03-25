package openrtb_ext

import (
	"encoding/json"
)

type ExtImpAdf struct {
	MasterTagID       json.Number `json:"mid,omitempty"`
	InventorySourceID int         `json:"inv,omitempty"`
	PlacementName     string      `json:"mname,omitempty"`
	PriceType         string      `json:"priceType,omitempty"`
}
