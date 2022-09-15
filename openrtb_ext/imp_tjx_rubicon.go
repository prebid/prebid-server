package openrtb_ext

import (
	"encoding/json"
)

// ExtImpTJXRubicon defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpTJXRubicon struct {
	AccountId int                   `json:"accountId"`
	SiteId    int                   `json:"siteId"`
	ZoneId    int                   `json:"zoneId"`
	Inventory json.RawMessage       `json:"inventory,omitempty"`
	Visitor   json.RawMessage       `json:"visitor,omitempty"`
	Video     tjxRubiconVideoParams `json:"video"`
	Debug     impExtTJXRubiconDebug `json:"debug,omitempty"`

	Region             string           `json:"region"`
	ViewabilityVendors []string         `json:"viewabilityvendors"`
	SKADNSupported     bool             `json:"skadn_supported"`
	MRAIDSupported     bool             `json:"mraid_supported"`
	BidFloor           *float64         `json:"bid_floor,omitempty"`
	Blocklist          RubiconBlocklist `json:"blocklist,omitempty"`
}
type RubiconBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.video
type tjxRubiconVideoParams struct {
	Language     string                      `json:"language,omitempty"`
	PlayerHeight int                         `json:"playerHeight,omitempty"`
	PlayerWidth  int                         `json:"playerWidth,omitempty"`
	VideoSizeID  int                         `json:"size_id,omitempty"`
	Skip         int                         `json:"skip,omitempty"`
	SkipDelay    int                         `json:"skipdelay,omitempty"`
	CompanionAd  tjxRubiconCompanionAdParams `json:"companion_ad,omitempty"`
}

type tjxRubiconCompanionAdParams struct {
	SizeID     int   `json:"size_id,omitempty"`
	AltSizeIDs []int `json:"alt_size_ids,omitempty"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.debug
type impExtTJXRubiconDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}
