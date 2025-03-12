package openrtb_ext

// ExtImpAdagio defines the contract for bidrequest.imp[i].ext.prebid.bidder for Adagio
type ExtImpAdagio struct {
	OrganizationID string `json:"organizationId"`
	Placement      string `json:"placement"`
	Pagetype       string `json:"pagetype,omitempty"`
	Category       string `json:"category,omitempty"`
}
