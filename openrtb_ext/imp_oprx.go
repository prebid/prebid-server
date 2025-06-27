package openrtb_ext

type ImpExtOprx struct {
	Key         string `json:"key"`
	PlacementID string `json:"placement_id"`
	Width       string `json:"width"`
	Height      string `json:"height"`
	BidFloor    string `json:"bid_floor"`
	Npi         string `json:"npi"`
	Ndc         string `json:"ndc"`
}
