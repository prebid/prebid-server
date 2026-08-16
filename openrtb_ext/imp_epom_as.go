package openrtb_ext

// ExtImpEpomAs defines the contract for bidrequest.imp[i].ext.prebid.bidder.epom_as
type ExtImpEpomAs struct {
	// Host is the serving host of the publisher's Epom Ad Server deployment.
	Host string `json:"host"`
	// PlacementKey identifies the placement within that deployment.
	PlacementKey string `json:"placementKey"`
	// Channel is a traffic-slice label used for targeting and reporting.
	Channel string `json:"channel,omitempty"`
	// CustomParams feed custom targeting and creative macros.
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}
