package openrtb_ext

import (
	"encoding/json"
)

type ExtImpTheadx struct {
	TagID             json.Number `json:"tagid,omitempty"`
	InventorySourceID int         `json:"wid,omitempty"`
	MemberID          int         `json:"pid,optional,omitempty"`
	PlacementName     string      `json:"pname,optional,omitempty"`
}
