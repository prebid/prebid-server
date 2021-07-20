package openrtb_ext

// ExtImpAarki defines the contract for bidrequest.imp[i].ext.aarki
type ExtImpAarki struct {
	Reward         int    `json:"reward"`
	Region         string `json:"region"`          // this field added to support multiple aarki endpoints
	SKADNSupported bool   `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool   `json:"mraid_supported"`
}
