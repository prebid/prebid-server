package openrtb_ext

import "encoding/json"

// ExtImpPubmatic defines the contract for bidrequest.imp[i].ext.pubmatic
// PublisherId and adSlot are mandatory parameters, others are optional parameters
// Keywords is bid specific parameter,
// WrapExt needs to be sent once per bid request

type ExtImpPubmatic struct {
	PublisherId string                  `json:"publisherId"`
	AdSlot      string                  `json:"adSlot"`
	WrapExt     json.RawMessage         `json:"wrapper,omitempty"`
	Keywords    []*ExtImpPubmaticKeyVal `json:"keywords,omitempty"`
}

// ExtImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.appnexus.keywords[i]
type ExtImpPubmaticKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
