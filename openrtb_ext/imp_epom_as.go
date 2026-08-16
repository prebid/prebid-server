package openrtb_ext

// ExtImpEpomAs defines the contract for bidrequest.imp[i].ext.prebid.bidder.epom_as
type ExtImpEpomAs struct {
	// Host is the serving host of the publisher's Epom Ad Server deployment.
	Host string `json:"host"`
	// PlacementKey identifies the placement within that deployment.
	PlacementKey string `json:"placementKey"`
}
