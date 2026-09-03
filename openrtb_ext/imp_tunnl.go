package openrtb_ext

// ExtImpTunnl defines the contract for bidrequest.imp[i].ext.prebid.bidder.tunnl
type ExtImpTunnl struct {
	// SID identifies the publisher to Tunnl. Tunnl issues these values, so the
	// adapter treats the sid as opaque and only passes it through to the
	// endpoint.
	SID string `json:"sid"`
}
