package openrtb_ext

// ExtImpMgid defines the contract for bidrequest.imp[i].ext.prebid.bidder.mgid
type ExtImpMgid struct {
	AccountId   string  `json:"accountId"`
	PlacementId string  `json:"placementId,omitempty"`
	Cur         string  `json:"cur"`
	Currency    string  `json:"currency"`
	BidFloor    float64 `json:"bidfloor"`
	BidFloor2   float64 `json:"bidFloor"`
}
