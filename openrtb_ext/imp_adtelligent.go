package openrtb_ext

import "encoding/json"

// ExtImpAdtelligent defines the contract for bidrequest.imp[i].ext.prebid.bidder.adtelligent
type ExtImpAdtelligent struct {
	SourceId    json.Number `json:"aid"`
	PlacementId int         `json:"placementId,omitempty"`
	SiteId      int         `json:"siteId,omitempty"`
	BidFloor    float64     `json:"bidFloor,omitempty"`
}
