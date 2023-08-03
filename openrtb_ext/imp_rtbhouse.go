package openrtb_ext

// ExtImpRTBHouse defines the contract for bidrequest.imp[i].ext.prebid.bidder.rtbhouse
type ExtImpRTBHouse struct {
	PublisherId string `json:"publisherId"`
	Region      string `json:"region"`

	BidFloor float64 `json:"bidfloor,omitempty"`
	Channel  string  `json:"channel,omitempty"`
}
