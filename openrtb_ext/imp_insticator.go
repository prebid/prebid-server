package openrtb_ext

// ExtImpInsticator defines the contract for bidrequest.imp[i].ext.prebid.bidder.insticator
type ExtImpInsticator struct {
	AdUnitId    string `json:"adUnitId,omitempty"`
	PublisherId string `json:"publisherId,omitempty"`
}
