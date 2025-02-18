package openrtb_ext

// ExtImpLoopme defines the contract for bidrequest.imp[i].ext.prebid.bidder.loopme
type ExtImpLoopme struct {
	PublisherId string `json:"publisherId"`
	BundleId    string `json:"bundleId"`
	PlacementId string `json:"placementId"`
}
