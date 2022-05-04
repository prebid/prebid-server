package openrtb_ext

// ExtImpJampp defines the contract for bidrequest.imp[i].ext.jampp
type ExtImpTJXJampp struct {
	Video          jamppVideoParams `json:"video"`
	Region         string           `json:"region"`
	SKADNSupported bool             `json:"skadn_supported"`
	MRAIDSupported bool             `json:"mraid_supported"`
	BidFloor       *float64         `json:"bid_floor,omitempty"`
}

// jamppVideoParams defines the contract for bidrequest.imp[i].ext.jampp.video
type jamppVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
