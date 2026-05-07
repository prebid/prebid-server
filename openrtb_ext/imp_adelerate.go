package openrtb_ext

type ImpExtAdelerate struct {
	PlacementId   string  `json:"placementId"`
	PublisherId   string  `json:"publisherId"`
	Floor         float64 `json:"floor,omitempty"`
	FloorCurrency string  `json:"floorCurrency,omitempty"`
}