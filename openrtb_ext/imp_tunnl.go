package openrtb_ext

// ExtImpTunnl defines the contract for bidrequest.imp[i].ext.prebid.bidder.tunnl
type ExtImpTunnl struct {
	// SID identifies the publisher and the Tunnl region the traffic belongs to.
	// Tunnl issues these values, so the adapter treats the sid as opaque apart
	// from its region, which must agree with the region the configured endpoint
	// points at.
	SID string `json:"sid"`
}
