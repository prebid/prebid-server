package openrtb_ext

// ExtImpTrustX2 defines the contract for bidrequest.imp[i].ext.prebid.bidder.trustx2
type ExtImpTrustX2 struct {
	PublisherId string `json:"publisher_id"`
	PlacementId string `json:"placement_id"`
}
