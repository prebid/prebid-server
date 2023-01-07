package openrtb_ext

// ExtImpSynacormedia defines the contract for bidrequest.imp[i].ext.prebid.bidder.synacormedia
type ExtImpSynacormedia struct {
	SeatId string `json:"seatId"`
	TagId  string `json:"tagId"`
}
