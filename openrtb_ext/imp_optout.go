package openrtb_ext

type ImpExtOptout struct {
	Publisher    string                 `json:"publisher"`
	AdSlot       string                 `json:"adSlot"`
	ID           *string                `json:"id,omitempty"`
	Customs      map[string]interface{} `json:"customs,omitempty"`
	IncludeOrtb2 bool                   `json:"includeOrtb2,omitempty"`
}
