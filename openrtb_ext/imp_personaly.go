package openrtb_ext

// ExtImpPersonaly defines the contract for bidrequest.imp[i].ext.personaly
type ExtImpPersonaly struct {
	Video          personalyVideoParams `json:"video"`
	SKADNSupported bool                 `json:"skadn_supported"`
	MRAIDSupported bool                 `json:"mraid_supported"`
}

// personalyVideoParams defines the contract for bidrequest.imp[i].ext.personaly.video
type personalyVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
