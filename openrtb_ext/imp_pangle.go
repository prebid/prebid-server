package openrtb_ext

type ImpExtPangle struct {
	Reward         int    `json:"reward"`
	SKADNSupported bool   `json:"skadn_supported"`
	MRAIDSupported bool   `json:"mraid_supported"`
	Token          string `json:"token"`
}
