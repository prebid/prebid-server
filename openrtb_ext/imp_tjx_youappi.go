package openrtb_ext

// ExtImpTJXYouAppi defines the contract for bidrequest.imp[i].ext.youappi
type ExtImpTJXYouAppi struct {
	Reward               int              `json:"reward"`
	Region               string           `json:"region"`          // this field added to support multiple youappi endpoints
	SKADNSupported       bool             `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported       bool             `json:"mraid_supported"`
	EndcardHTMLSupported bool             `json:"endcard_html_supported"`
	BidFloor             *float64         `json:"bid_floor,omitempty"`
	Blocklist            YouAppiBlocklist `json:"blocklist,omitempty"`
}
type YouAppiBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}
