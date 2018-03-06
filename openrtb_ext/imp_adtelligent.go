package openrtb_ext

// ExtImpAdtelligent defines the contract for bidrequest.imp[i].ext.adtelligent
type ExtImpAdtelligent struct {
	SourceId    int     `json:"aid"`
	PlacementId int     `json:"placementId,omitempty"`
	SiteId      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
