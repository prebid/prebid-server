package openrtb_ext

// ExtImpOrbidder defines the contract for bidrequest.imp[i].ext.openx
type ExtImpOrbidder struct {
	AccountId   string  `json:"accountId"`
	PlacementId string  `json:"placementId"`
	BidFloor    float64 `json:"bidfloor"`
}
