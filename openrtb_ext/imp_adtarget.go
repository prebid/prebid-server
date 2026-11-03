package openrtb_ext

import "encoding/json"

// ExtImpAdtarget defines the contract for bidrequest.imp[i].ext.prebid.bidder.adtarget
type ExtImpAdtarget struct {
	SourceId    json.Number `json:"aid"`
	PlacementId int         `json:"placementId,omitempty"`
	SiteId      int         `json:"siteId,omitempty"`
	BidFloor    float64     `json:"bidFloor,omitempty"`
}
