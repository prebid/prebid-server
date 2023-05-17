package openrtb_ext

// ExtImpImds defines the contract for bidrequest.imp[i].ext.prebid.bidder.imds
type ExtImpImds struct {
	SeatId string `json:"seatId"`
	TagId  string `json:"tagId"`
}
