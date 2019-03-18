package openrtb_ext

import (
	"encoding/json"
)

// ExtImpRubicon defines the contract for bidrequest.imp[i].ext.rubicon
type ExtImpRubicon struct {
	AccountId int                `json:"accountId"`
	SiteId    int                `json:"siteId"`
	ZoneId    int                `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory"`
	Visitor   json.RawMessage    `json:"visitor"`
	Video     rubiconVideoParams `json:"video"`
}

// rubiconVideoParams defines the contract for bidrequest.imp[i].ext.rubicon.video
type rubiconVideoParams struct {
	Language     string `json:"language,omitempty"`
	PlayerHeight int    `json:"playerHeight,omitempty"`
	PlayerWidth  int    `json:"playerWidth,omitempty"`
	VideoSizeID  int    `json:"size_id,omitempty"`
	Skip         int    `json:"skip,omitempty"`
	SkipDelay    int    `json:"skipdelay,omitempty"`
}
