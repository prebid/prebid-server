package openrtb_ext

// ExtImpTaurusX defines the contract for bidrequest.imp[i].ext.taurusx
type ExtImpTaurusX struct {
	Reward         int    `json:"reward"`
	Region         string `json:"region"`          // this field added to support multiple taurusx endpoints
	SKADNSupported bool   `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool   `json:"mraid_supported"`
}
