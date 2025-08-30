package openrtb_ext

// ExtImpDXTech defines the contract for bidrequest.imp[i].ext.prebid.bidder.dxtech
type ExtImpDXTech struct {
	PublisherId string `json:"publisherId"`
	PlacementId string `json:"placementId"`
}
