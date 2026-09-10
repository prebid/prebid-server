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
}
