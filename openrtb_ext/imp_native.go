package openrtb_ext

import (
	"encoding/json"
)

// ExtImpNative defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpNative struct {
	AccountId json.Number        `json:"accountId"`
	SiteId    json.Number        `json:"siteId"`
	ZoneId    json.Number        `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory,omitempty"`
	Keywords  []string           `json:"keywords,omitempty"`
	Visitor   json.RawMessage    `json:"visitor,omitempty"`
	Debug     impExtRubiconDebug `json:"debug,omitempty"`
}

// nativeDebugParams defines the contract for bidrequest.imp[i].ext.rubicon.debug
type impExtNativeDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}
