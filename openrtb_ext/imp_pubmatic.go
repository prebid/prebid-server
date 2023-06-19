package openrtb_ext

import "encoding/json"

// ExtImpPubmatic defines the contract for bidrequest.imp[i].ext.prebid.bidder.pubmatic
// PublisherId is mandatory parameters, others are optional parameters
// AdSlot is identifier for specific ad placement or ad tag
// Keywords is bid specific parameter,
// WrapExt needs to be sent once per bid request

type ExtImpPubmatic struct {
	PublisherId string                  `json:"publisherId"`
	AdSlot      string                  `json:"adSlot"`
	Dctr        string                  `json:"dctr"`
	PmZoneID    string                  `json:"pmzoneid"`
	WrapExt     json.RawMessage         `json:"wrapper,omitempty"`
	Keywords    []*ExtImpPubmaticKeyVal `json:"keywords,omitempty"`
	Kadfloor    string                  `json:"kadfloor,omitempty"`
}

// ExtImpPubmaticKeyVal defines the contract for bidrequest.imp[i].ext.prebid.bidder.pubmatic.keywords[i]
type ExtImpPubmaticKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
