package openrtb_ext

// ImpExtCwire defines the contract for MakeRequests `request.imp[i].ext.bidder`
type ImpExtCWire struct {
	PlacementID int     `json:"placementId,omitempty"`
	SiteID      int     `json:"siteId,omitempty"`
	PageViewID  float64 `json:"pageViewId,omitempty"`
	CreativeID  string  `json:"creativeId,omitempty"`
}
