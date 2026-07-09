package openrtb_ext

// ExtImpRealry is the bidder-specific imp ext passed by publishers in
// `imp.ext.prebid.bidder.realry`. The adapter forwards the BidRequest
// as-is to bid.realry.com, so publishers don't strictly need to set
// these — they're there so a publisher can pin a placement to a known
// account on the Realry side for reporting / quality assignment.
type ExtImpRealry struct {
	// PlacementId is the publisher-side identifier for the slot. Forwarded
	// to the Realry bidder so impression/click logs can be diced per
	// publisher placement without needing to parse site.id.
	PlacementId string `json:"placementId"`

	// SellerId is an optional realry-side advertiser id the publisher
	// has been onboarded against. Omit unless explicitly assigned by
	// the Realry partnerships team.
	SellerId string `json:"sellerId,omitempty"`
}
