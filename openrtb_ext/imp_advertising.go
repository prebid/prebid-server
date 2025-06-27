package openrtb_ext

// ExtImpAdvertising defines the contract for bidrequest.imp[i].ext.prebid.bidder.advertising
type ExtImpAdvertising struct {
	SeatId string `json:"seatId"`
	TagId  string `json:"tagId"`
}
