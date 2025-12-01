package scalibur

// ExtImpScalibur mirrors the bidder params from Prebid.js.
type ExtImpScalibur struct {
	PlacementID string   `json:"placementId"`           // required
	BidFloor    *float64 `json:"bidfloor,omitempty"`    // optional, used as fallback
	BidFloorCur string   `json:"bidfloorcur,omitempty"` // optional, defaults to USD if empty
}
