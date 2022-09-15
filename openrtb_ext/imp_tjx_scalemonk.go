package openrtb_ext

// ExtImpScaleMonk defines the contract for bidrequest.imp[i].ext.scalemonk
type ExtImpTJXScaleMonk struct {
	Video          scaleMonkVideoParams `json:"video"`
	Region         string               `json:"region"`
	SKADNSupported bool                 `json:"skadn_supported"`
	MRAIDSupported bool                 `json:"mraid_supported"`
	BidFloor       *float64             `json:"bid_floor,omitempty"`
	Blocklist      ScaleMonkBlocklist   `json:"blocklist,omitempty"`
}
type ScaleMonkBlocklist struct {
	BApp []string `json:"bapp,omitempty"`
	BAdv []string `json:"badv,omitempty"`
}

// scaleMonkVideoParams defines the contract for bidrequest.imp[i].ext.scalemonk.video
type scaleMonkVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
