package openrtb_ext

type ImpExtPangle struct {
	Token       string `json:"token"`
	AppID       string `json:"appid,omitempty"`
	PlacementID string `json:"placementid,omitempty"`

	Reward         int      `json:"reward"`
	Region         string   `json:"region"`
	SKADNSupported bool     `json:"skadn_supported"`
	MRAIDSupported bool     `json:"mraid_supported"`
	BidFloor       *float64 `json:"bid_floor,omitempty"`
}
