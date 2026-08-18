package openrtb_ext

// ImpExtRevantage defines the contract for the bidder-specific portion of
// imp.ext when targeting the Revantage adapter.
//
// feedId is required and identifies the publisher feed the impression
// belongs to. placementId and publisherId are optional pass-through values.
type ImpExtRevantage struct {
	FeedID      string `json:"feedId"`
	PlacementID string `json:"placementId,omitempty"`
	PublisherID string `json:"publisherId,omitempty"`
}
