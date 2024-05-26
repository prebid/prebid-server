package openrtb_ext

// ExtImpOpenWeb defines the contract for bidrequest.imp[i].ext.prebid.bidder.openweb
type ExtImpOpenWeb struct {
	Aid         int    `json:"aid"`
	Org         string `json:"org"`
	PlacementID string `json:"placementId,omitempty"`
}
