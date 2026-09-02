package openrtb_ext

// ImpExtPeak226 defines the contract for bidrequest.imp[i].ext.prebid.bidder.peak226
type ImpExtPeak226 struct {
	PublisherID string `json:"publisherId"`
	PlacementID string `json:"placementId"`
	Region      string `json:"region,omitempty"`
}
