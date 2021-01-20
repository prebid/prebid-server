package openrtb_ext

// ExtImpCrossInstall defines the contract for bidrequest.imp[i].ext.crossinstall
type ExtImpCrossInstall struct {
	Reward         int    `json:"reward"`
	Region         string `json:"region"`          // this field added to support multiple crossinstall endpoints
	SKADNSupported bool   `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool   `json:"mraid_supported"`
}
