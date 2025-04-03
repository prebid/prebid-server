package openrtb_ext

type ImpExtOptidigital struct {
	PublisherID  string `json:"publisherId"`
	PlacementID  string `json:"placementId"`
	PageTemplate string `json:"pageTemplate,omitempty"`
	DivID        string `json:"divId,omitempty"`
}
