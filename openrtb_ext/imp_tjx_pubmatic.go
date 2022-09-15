package openrtb_ext

import "encoding/json"

// ExtImpPubmatic defines the contract for bidrequest.imp[i].ext.pubmatic
// PublisherId is mandatory parameters, others are optional parameters
// AdSlot is identifier for specific ad placement or ad tag
// Keywords is bid specific parameter,
// WrapExt needs to be sent once per bid request

type ExtImpTJXPubmatic struct {
	PublisherId string                     `json:"publisherId"`
	AdSlot      string                     `json:"adSlot"`
	Dctr        string                     `json:"dctr"`
	PmZoneID    string                     `json:"pmzoneid"`
	WrapExt     json.RawMessage            `json:"wrapper,omitempty"`
	Keywords    []*ExtImpTJXPubmaticKeyVal `json:"keywords,omitempty"`

	Reward         int               `json:"reward"`
	SiteID         int               `json:"site_id"`
	Region         string            `json:"region"`
	SKADNSupported bool              `json:"skadn_supported"`
	MRAIDSupported bool              `json:"mraid_supported"`
	BidFloor       *float64          `json:"bid_floor,omitempty"`
	Blocklist      PubmaticBlocklist `json:"blocklist,omitempty"`
}
type PubmaticBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}

// ExtImpTJXPubmaticKeyVal defines the contract for bidrequest.imp[i].ext.pubmatic.keywords[i]
type ExtImpTJXPubmaticKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
