package openrtb_ext

// ExtImpAppier defines the contract for bidrequest.imp[i].ext.appier
type ExtImpTJXAppier struct {
	Video                appierVideoParams `json:"video"`
	Region               string            `json:"region"`
	SKADNSupported       bool              `json:"skadn_supported"`
	MRAIDSupported       bool              `json:"mraid_supported"`
	EndcardHTMLSupported bool              `json:"endcard_html_supported"`
	BidFloor             *float64          `json:"bid_floor,omitempty"`
}

// appierVideoParams defines the contract for bidrequest.imp[i].ext.appier.video
type appierVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
