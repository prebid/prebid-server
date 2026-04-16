package openrtb_ext

type ExtImpStackAdapt struct {
	PublisherId string                  `json:"publisherId"`
	SupplyId    string                  `json:"supplyId"`
	PlacementId string                  `json:"placementId,omitempty"`
	Banner      *ExtImpStackAdaptBanner `json:"banner,omitempty"`
	BidFloor    float64                 `json:"bidfloor,omitempty"`
}

type ExtImpStackAdaptBanner struct {
	ExpDir []int `json:"expdir,omitempty"`
}
