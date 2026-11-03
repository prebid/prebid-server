package openrtb_ext

import "encoding/json"

// ExtImpBidmatic defines the contract for bidrequest.imp[i].ext.prebid.bidder.bidmatic
type ExtImpBidmatic struct {
	SourceId    json.Number `json:"source"`
	PlacementId int         `json:"placementId,omitempty"`
	SiteId      int         `json:"siteId,omitempty"`
	BidFloor    float64     `json:"bidFloor,omitempty"`
}
