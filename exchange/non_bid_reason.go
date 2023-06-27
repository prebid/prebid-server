package exchange

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md
type NonBidReason int

const (
	NoBidUnknownError                      NonBidReason = 0   // No Bid - General
	ResponseRejectedCategoryMappingInvalid NonBidReason = 303 // Response Rejected - Category Mapping Invalid
)

// Ptr returns pointer to own value.
func (n NonBidReason) Ptr() *NonBidReason {
	return &n
}

// Val safely dereferences pointer, returning default value (NoBidUnknownError) for nil.
func (n *NonBidReason) Val() NonBidReason {
	if n == nil {
		return NoBidUnknownError
	}
	return *n
}
