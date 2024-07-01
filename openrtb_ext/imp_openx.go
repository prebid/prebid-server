package openrtb_ext

import "encoding/json"

// ExtImpOpenx defines the contract for bidrequest.imp[i].ext.prebid.bidder.openx
type ExtImpOpenx struct {
	Unit         json.Number            `json:"unit"`
	Platform     string                 `json:"platform"`
	DelDomain    string                 `json:"delDomain"`
	CustomFloor  json.Number            `json:"customFloor"`
	CustomParams map[string]interface{} `json:"customParams"`
}
