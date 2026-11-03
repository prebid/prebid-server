package openrtb_ext

// ExtImpOpenWeb defines the contract for bidrequest.imp[i].ext.prebid.bidder.openweb
type ExtImpOpenWeb struct {
	Aid         int    `json:"aid,omitempty"`
	Org         string `json:"org,omitempty"`
	PlacementID string `json:"placementId"`
}
