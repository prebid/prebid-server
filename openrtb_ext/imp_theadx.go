package openrtb_ext

import (
	"encoding/json"
)

type ExtImpTheadx struct {
	MasterTagID       json.Number `json:"tagid,omitempty"`
	InventorySourceID int         `json:"inv,omitempty"`
	PlacementName     string      `json:"mname,omitempty"`
	PriceType         string      `json:"priceType,omitempty"`
}
