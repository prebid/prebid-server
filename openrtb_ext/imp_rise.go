package openrtb_ext

// ImpExtRise defines the contract for bidrequest.imp[i].ext.prebid.bidder.rise
type ImpExtRise struct {
	PublisherID string `json:"publisher_id"`
	Org         string `json:"org"`
	PlacementID string `json:"placementId"`
}
