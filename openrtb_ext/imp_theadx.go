package openrtb_ext

import (
	"encoding/json"
)

type ExtImpTheadx struct {
	TagID             json.Number `json:"tagid"`
	InventorySourceID int         `json:"wid,omitempty"`
	MemberID          int         `json:"pid,omitempty"`
	PlacementName     string      `json:"pname,omitempty"`
}
