package openrtb_ext

// ExtImpContxtful defines the contract for bidrequest.imp[i].ext.prebid.bidder.contxtful
type ExtImpContxtful struct {
	PlacementId string `json:"placementId"`
	CustomerId  string `json:"customerId"`
}
