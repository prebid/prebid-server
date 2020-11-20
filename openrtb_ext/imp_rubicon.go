package openrtb_ext

import (
	"encoding/json"
)

// ExtImpRubicon defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpRubicon struct {
	AccountId          int                `json:"accountId"`
	SiteId             int                `json:"siteId"`
	ZoneId             int                `json:"zoneId"`
	Inventory          json.RawMessage    `json:"inventory,omitempty"`
	Visitor            json.RawMessage    `json:"visitor,omitempty"`
	Video              rubiconVideoParams `json:"video"`
	Debug              impExtRubiconDebug `json:"debug,omitempty"`
	Region             string             `json:"region"`
	ViewabilityVendors []string           `json:"viewabilityvendors"`
	SKADNSupported     bool               `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported     bool               `json:"mraid_supported"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.video
type rubiconVideoParams struct {
	Language     string                   `json:"language,omitempty"`
	PlayerHeight int                      `json:"playerHeight,omitempty"`
	PlayerWidth  int                      `json:"playerWidth,omitempty"`
	VideoSizeID  int                      `json:"size_id,omitempty"`
	Skip         int                      `json:"skip,omitempty"`
	SkipDelay    int                      `json:"skipdelay,omitempty"`
	CompanionAd  rubiconCompanionAdParams `json:"companion_ad,omitempty"`
}

type rubiconCompanionAdParams struct {
	SizeID     int   `json:"size_id,omitempty"`
	AltSizeIDs []int `json:"alt_size_ids,omitempty"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.debug
type impExtRubiconDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}
