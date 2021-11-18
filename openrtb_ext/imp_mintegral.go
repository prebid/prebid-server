package openrtb_ext

// ExtImpMintegral defines the contract for bidrequest.imp[i].ext.mintegral
type ExtImpMintegral struct {
	Video          mintegralVideoParams `json:"video"`
	Region         string               `json:"region"`
	SKADNSupported bool                 `json:"skadn_supported"`
	MRAIDSupported bool                 `json:"mraid_supported"`
}

// mintegralVideoParams defines the contract for bidrequest.imp[i].ext.mintegral.video
type mintegralVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
