package openrtb_ext

// ExtImpRiptide defines the contract for bidrequest.imp[i].ext.prebid.bidder.riptide (and for
// every alias of it): the publisher's and the placement's public ids, and an optional floor in
// USD. The serving host is not a param: it is the bidder's endpoint in its bidder-info file.
type ExtImpRiptide struct {
	PublisherID string  `json:"publisherId"`
	PlacementID string  `json:"placementId"`
	BidFloor    float64 `json:"bidfloor,omitempty"`
}
