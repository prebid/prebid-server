package openrtb_ext

// ImpExtCwire defines the contract for MakeRequests `request.imp[i].ext.bidder`
type ImpExtCWire struct {
	PlacementID int     `json:"placementId"`
	SiteID      int     `json:"siteId"`
	PageViewID  float64 `json:"pageViewId,omitempty"`
}
