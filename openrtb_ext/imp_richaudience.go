package openrtb_ext

type ExtImpRichaudience struct {
	Pid         string  `json:"pid"`
	SupplyType  string  `json:"supplyType"`
	BidFloor    float64 `json:"bid_floor"`
	BidFloorCur string  `json:"bid_floor_cur"`
	TestRa      bool    `json:"testra"`
}
