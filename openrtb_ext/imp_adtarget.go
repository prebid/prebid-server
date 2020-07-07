package openrtb_ext

// ExtImpAdtarget defines the contract for bidrequest.imp[i].ext.adtarget
type ExtImpAdtarget struct {
	SourceId    int     `json:"aid"`
	PlacementId int     `json:"placementId,omitempty"`
	SiteId      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
