package openrtb_ext

type ExtImpBetween struct {
	Host        string  `json:"host"`
	PublisherID string  `json:"publisher_id"`
	BidFloor    float64 `json:"bid_floor"`
	BidFloorCur string  `json:"bid_floor_cur"`
}
