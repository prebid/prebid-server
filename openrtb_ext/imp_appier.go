package openrtb_ext

// ExtImpAppier defines the contract for bidrequest.imp[i].ext.appier
type ExtImpAppier struct {
	Video          appierVideoParams `json:"video"`
	Region         string            `json:"region"`
	SKADNSupported bool              `json:"skadn_supported"`
	MRAIDSupported bool              `json:"mraid_supported"`
}

// appierVideoParams defines the contract for bidrequest.imp[i].ext.appier.video
type appierVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
