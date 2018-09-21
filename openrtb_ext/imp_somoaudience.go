package openrtb_ext

type ExtImpSomoaudience struct {
	PlacementHash string  `json:"placement_hash"`
	BidFloor      float64 `json:"bid_floor,omitempty"`
}
