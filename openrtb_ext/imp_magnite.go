package openrtb_ext

import (
	"encoding/json"
)

// ExtImpMagnite defines the contract for bidrequest.imp[i].ext.prebid.bidder.magnite
type ExtImpMagnite struct {
	AccountId        json.Number        `json:"accountId"`
	SiteId           json.Number        `json:"siteId"`
	ZoneId           json.Number        `json:"zoneId"`
	Inventory        json.RawMessage    `json:"inventory,omitempty"`
	BidOnMultiformat bool               `json:"bidonmultiformat,omitempty"`
	Keywords         []string           `json:"keywords,omitempty"`
	Visitor          json.RawMessage    `json:"visitor,omitempty"`
	Video            impExtMagniteVideo `json:"video"`
	Debug            impExtMagniteDebug `json:"debug,omitempty"`
}

// impExtMagniteVideo defines the contract for bidrequest.imp[i].ext.prebid.bidder.magnite.video
type impExtMagniteVideo struct {
	Language     string      `json:"language,omitempty"`
	PlayerHeight json.Number `json:"playerHeight,omitempty"`
	PlayerWidth  json.Number `json:"playerWidth,omitempty"`
	VideoSizeID  int         `json:"size_id,omitempty"`
	Skip         int         `json:"skip,omitempty"`
	SkipDelay    int         `json:"skipdelay,omitempty"`
}

// impExtMagniteDebug defines the contract for bidrequest.imp[i].ext.prebid.bidder.magnite.debug
type impExtMagniteDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}
