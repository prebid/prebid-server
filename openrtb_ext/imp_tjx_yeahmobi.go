package openrtb_ext

type ExtImpTJXYeahMobi struct {
	PlacementID    string            `json:"placementid"`
	Blocklist      YeahMobiBlocklist `json:"blocklist,omitempty"`
	SKADNSupported bool              `json:"skadn_supported"`
	MRAIDSupported bool              `json:"mraid_supported"`
	Region         string            `json:"region"`
	BidFloor       *float64          `json:"bid_floor,omitempty"`
}
type YeahMobiBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}
