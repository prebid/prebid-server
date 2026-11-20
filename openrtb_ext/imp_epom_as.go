package openrtb_ext

// ExtImpEpomAs defines the contract for bidrequest.imp[i].ext.prebid.bidder.epom_as
type ExtImpEpomAs struct {
	// NetworkID names the Epom network ("n" and the network number); the request goes to
	// https://{NetworkID}.eashb.com/hb/bid.
	NetworkID string `json:"networkId"`
	// Host is the network's own serving domain. Prebid.js sends browser bids there so the
	// network's identity cookie comes along; a server request carries no browser cookie, so this
	// adapter accepts it for parameter parity and does not use it.
	Host string `json:"host,omitempty"`
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
