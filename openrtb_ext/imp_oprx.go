package openrtb_ext

type ImpExtOprx struct {
	Key         string  `json:"key"`
	PlacementID int     `json:"placement_id"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	BidFloor    float64 `json:"bid_floor"`
	Npi         string  `json:"npi"`
	Ndc         string  `json:"ndc"`
	Type        string  `json:"type"`
}
