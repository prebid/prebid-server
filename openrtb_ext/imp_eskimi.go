package openrtb_ext

// ExtImpEskimi defines the contract for bidrequest.imp[i].ext.prebid.bidder.eskimi.
// COPPA and test flags are handled by PBS core via request.regs.coppa / request.test;
// they are not duplicated as bidder params.
type ExtImpEskimi struct {
	PlacementID int64    `json:"placementId"`
	BidFloor    float64  `json:"bidFloor,omitempty"`
	BidFloorCur string   `json:"bidFloorCur,omitempty"`
	Bcat        []string `json:"bcat,omitempty"`
	Badv        []string `json:"badv,omitempty"`
	Bapp        []string `json:"bapp,omitempty"`
	Battr       []int64  `json:"battr,omitempty"`
}
