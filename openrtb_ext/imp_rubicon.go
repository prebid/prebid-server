package openrtb_ext

import (
	"encoding/json"
)

// ExtImpRubicon defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpRubicon struct {
	AccountId json.Number        `json:"accountId"`
	SiteId    json.Number        `json:"siteId"`
	ZoneId    json.Number        `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory,omitempty"`
	Keywords  []string           `json:"keywords,omitempty"`
	Visitor   json.RawMessage    `json:"visitor,omitempty"`
	Video     rubiconVideoParams `json:"video"`
	Debug     impExtRubiconDebug `json:"debug,omitempty"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.video
type rubiconVideoParams struct {
	Language     string      `json:"language,omitempty"`
	PlayerHeight json.Number `json:"playerHeight,omitempty"`
	PlayerWidth  json.Number `json:"playerWidth,omitempty"`
	VideoSizeID  int         `json:"size_id,omitempty"`
	Skip         int         `json:"skip,omitempty"`
	SkipDelay    int         `json:"skipdelay,omitempty"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.debug
type impExtRubiconDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}
