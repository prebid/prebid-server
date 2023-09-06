package openrtb_ext

type ExtImpUndertone struct {
	PublisherID int `json:"publisherId"`
	PlacementID int `json:"placementId"`
}

type UndertoneImpExt struct {
	Gpid string `json:"gpid,omitempty"`
}
