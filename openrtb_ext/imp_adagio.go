package openrtb_ext

type ExtImpAdagio struct {
	OrganizationID string `json:"organizationId"`
	Placement      string `json:"placement"`
	Pagetype       string `json:"pagetype,omitempty"`
	Category       string `json:"category,omitempty"`
}
