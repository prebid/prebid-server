package openrtb_ext

// ExtImpUnicorn defines the contract for bidrequest.imp[i].ext.unicorn
type ExtImpUnicorn struct {
	Reward         int    `json:"reward"`
	Region         string `json:"region"`          // this field added to support multiple unicorn endpoints
	SKADNSupported bool   `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool   `json:"mraid_supported"`
}
