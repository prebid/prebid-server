package openrtb_ext

// ExtImpTaurusX defines the contract for bidrequest.imp[i].ext.taurusx
type ExtImpTJXTaurusX struct {
	Reward         int              `json:"reward"`
	Region         string           `json:"region"`          // this field added to support multiple taurusx endpoints
	SKADNSupported bool             `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool             `json:"mraid_supported"`
	BidFloor       *float64         `json:"bid_floor,omitempty"`
	Blocklist      TaurusxBlocklist `json:"blocklist,omitempty"`
}
type TaurusxBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}
