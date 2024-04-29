package openrtb_ext

type ImpExtConcert struct {
	PartnerId   string   `json:"partnerId"`
	PlacementId *int     `json:"placementId,omitempty"`
	Site        *string  `json:"site,omitempty"`
	Slot        *string  `json:"slot,omitempty"`
	Sizes       *[][]int `json:"sizes,omitempty"`
}
