package openrtb_ext

// ExtImpCwire defines the contract for bidrequest.imp[i].ext.cwire
type ExtImpCWire struct {
	SourceId    int     `json:"aid"`
	PlacementId int     `json:"placementId,omitempty"`
	SiteId      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
