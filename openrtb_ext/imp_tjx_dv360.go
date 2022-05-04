package openrtb_ext

type ExtImpTJXDV360 struct {
	Region         string   `json:"region"`
	Reward         int      `json:"reward"`
	SKADNSupported bool     `json:"skadn_supported"`
	MRAIDSupported bool     `json:"mraid_supported"`
	RawIP          string   `json:"raw_ip"`
	PubID          string   `json:"pub_id"`
	BidFloor       *float64 `json:"bid_floor,omitempty"`
}
