package openrtb_ext

import (
	"encoding/json"
)

// ExtImpRubicon defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpRubicon struct {
	AccountId int             `json:"accountId"`
	SiteId    int             `json:"siteId"`
	ZoneId    int             `json:"zoneId"`
	Inventory json.RawMessage `json:"inventory"`
	Visitor   json.RawMessage `json:"visitor"`
}