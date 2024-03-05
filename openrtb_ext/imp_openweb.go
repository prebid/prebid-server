package openrtb_ext

// ExtImpOpenWeb defines the contract for bidrequest.imp[i].ext.prebid.bidder.openweb
type ExtImpOpenWeb struct {
	SourceID    int     `json:"aid"`
	PlacementID int     `json:"placementId,omitempty"`
	SiteID      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
