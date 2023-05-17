package openrtb_ext

// ExtImpOpenx defines the contract for bidrequest.imp[i].ext.prebid.bidder.openx
type ExtImpOpenx struct {
	Unit         string                 `json:"unit"`
	Platform     string                 `json:"platform"`
	DelDomain    string                 `json:"delDomain"`
	CustomFloor  float64                `json:"customFloor"`
	CustomParams map[string]interface{} `json:"customParams"`
}
