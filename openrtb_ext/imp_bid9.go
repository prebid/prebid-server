package openrtb_ext

// ExtImpBid9 defines the contract for bidrequest.imp[i].ext.prebid.bidder.bid9
type ExtImpBid9 struct {
	ChannelID   string  `json:"channelId"`
	PublisherID string  `json:"publisherId,omitempty"`
	BidFloor    float64 `json:"bidFloor,omitempty"`
	BidFloorCur string  `json:"bidFloorCur,omitempty"`
}
