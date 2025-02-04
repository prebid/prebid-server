package openrtb_ext

import "encoding/json"

// ExtImpAniview defines the contract for bidrequest.imp[i].ext.prebid.bidder.aniview
// PublisherId is mandatory parameters, others are optional parameters
// AdSlot is identifier for specific ad placement or ad tag
// Keywords is bid specific parameter,
// WrapExt needs to be sent once per bid request

type ExtImpAniview struct {
	PublisherId string                 `json:"publisherId"`
	AdSlot      string                 `json:"adSlot"`
	Dctr        string                 `json:"dctr"`
	PmZoneID    string                 `json:"pmzoneid"`
	WrapExt     json.RawMessage        `json:"wrapper,omitempty"`
	Keywords    []*ExtImpAniviewKeyVal `json:"keywords,omitempty"`
	Kadfloor    string                 `json:"kadfloor,omitempty"`
}

// ExtImpAniviewKeyVal defines the contract for bidrequest.imp[i].ext.prebid.bidder.aniview.keywords[i]
type ExtImpAniviewKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
