package openrtb_ext

/*
Create a file with the path openrtb_ext/imp_{bidder}.go containing an exported
(must start with an upper case letter) data structure named ImpExt{Bidder}. All
required and optional bidder parameters from the JSON Schema should be
represented as fields
*/
// Adtelligent comment: ExtImpCwire defines the contract for bidrequest.imp[i].ext.cwire
type ImpExtCWire struct {
	SourceId    int     `json:"aid"`
	PlacementId int     `json:"placementId,omitempty"`
	SiteId      int     `json:"siteId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
}
