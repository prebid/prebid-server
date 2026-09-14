package openrtb_ext

type ImpExtAdelerate struct {
	PlacementID   string  `json:"placementId"`
	PublisherID   string  `json:"publisherId"`
	Floor         float64 `json:"floor,omitempty"`
	FloorCurrency string  `json:"floorCurrency,omitempty"`
}
