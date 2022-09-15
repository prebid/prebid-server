package openrtb_ext

// ExtImpTJXAarki defines the contract for bidrequest.imp[i].ext.aarki
type ExtImpTJXAarki struct {
	Reward               int            `json:"reward"`
	Region               string         `json:"region"`          // this field added to support multiple aarki endpoints
	SKADNSupported       bool           `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported       bool           `json:"mraid_supported"`
	EndcardHTMLSupported bool           `json:"endcard_html_supported"`
	BidFloor             *float64       `json:"bid_floor,omitempty"`
	Blocklist            AarkiBlocklist `json:"blocklist,omitempty"`
}
type AarkiBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}
