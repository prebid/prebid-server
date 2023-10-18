package openrtb_ext

// ImpExtSilverpush defines the contract for bidrequest.imp[i].ext.prebid.bidder.silverpush
// PublisherId  is mandatory parameters
type ImpExtSilverpush struct {
	PublisherId string  `json:"publisherId"`
	BidFloor    float64 `json:"bidfloor"`
}
