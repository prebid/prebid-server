package openrtb_ext

type ImpExtPangle struct {
	Token       string `json:"token"`
	AppID       string `json:"appid,omitempty"`
	PlacementID string `json:"placementid,omitempty"`

	Reward         int  `json:"reward"`
	SKADNSupported bool `json:"skadn_supported"`
	MRAIDSupported bool `json:"mraid_supported"`
}
