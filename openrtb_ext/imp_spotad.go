package openrtb_ext

// ExtImpSpotAd defines the contract for bidrequest.imp[i].ext.spotad
type ExtImpSpotAd struct {
	Video          spotAdVideoParams `json:"video"`
	Region         string            `json:"region"`
	SKADNSupported bool              `json:"skadn_supported"`
	MRAIDSupported bool              `json:"mraid_supported"`
	BidFloor       *float64          `json:"bid_floor,omitempty"`
}

// spotadVideoParams defines the contract for bidrequest.imp[i].ext.spotad.video
type spotAdVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
