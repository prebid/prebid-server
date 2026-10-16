package openrtb_ext

// ExtImpOcm defines the contract for bidrequest.imp[i].ext.prebid.bidder.ocm.
type ExtImpOcm struct {
	// PublisherID is the Orange Click Media publisher identifier. It is propagated
	// to the request-level publisher ID, so every impression in a single request
	// must carry the same value.
	PublisherID string `json:"publisherId"`

	// PlacementID names the impression's stored request on the OCM endpoint, which
	// expands it into the impression's real configuration. It is required: an
	// impression that references no stored request cannot be resolved.
	PlacementID string `json:"placementId"`

	// TmaxBufferMs is deducted from the request tmax before it is forwarded. The
	// OCM endpoint is itself a Prebid Server that would otherwise spend the whole
	// budget on its own bidders, leaving no room for the round trip back. Nil
	// selects the adapter default; an explicit 0 disables the buffer.
	TmaxBufferMs *int `json:"tmaxBufferMs,omitempty"`
}
