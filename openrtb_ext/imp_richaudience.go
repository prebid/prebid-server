package openrtb_ext

type ExtImpRichaudience struct {
	Pid         string  `json:"pid"`
	SupplyType  string  `json:"supplyType"`
	BidFloor    float64 `json:"bidfloor"`
	BidFloorCur string  `json:"bidfloorcur"`
	Test        bool    `json:"test"`
}
