package openrtb_ext

// ExtImpOpenWeb defines the contract for bidrequest.imp[i].ext.openweb
type ExtImpOpenWeb struct {
	SourceId    int     `json:"aid"`
	PlacementId int     `json:"placementId,omitempty"`
	SiteId      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
