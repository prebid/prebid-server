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
	// BidFloor is a CPM floor applied only when the request carries no floor of
	// its own, so a Price Floors module result always wins.
	BidFloor float64 `json:"bidFloor,omitempty"`
	// BidFloorCur is the currency of BidFloor, defaulting to USD.
	BidFloorCur string `json:"bidFloorCur,omitempty"`
}
