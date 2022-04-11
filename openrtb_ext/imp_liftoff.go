package openrtb_ext

// ExtImpLiftoff defines the contract for bidrequest.imp[i].ext.liftoff
type ExtImpLiftoff struct {
	Video                liftoffVideoParams `json:"video"`
	Region               string             `json:"region"`
	SKADNSupported       bool               `json:"skadn_supported"`
	MRAIDSupported       bool               `json:"mraid_supported"`
	EndcardHTMLSupported bool               `json:"endcard_html_supported"`
	BidFloor             *float64           `json:"bid_floor,omitempty"`
}

// liftoffVideoParams defines the contract for bidrequest.imp[i].ext.liftoff.video
type liftoffVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
