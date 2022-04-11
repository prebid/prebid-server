package openrtb_ext

// ExtImpKadenAI defines the contract for bidrequest.imp[i].ext.kadenai
type ExtImpKadenAI struct {
	Video          kadenaiVideoParams `json:"video"`
	SKADNSupported bool               `json:"skadn_supported"`
	MRAIDSupported bool               `json:"mraid_supported"`
	BidFloor       *float64           `json:"bid_floor,omitempty"`
}

// kadenaiVideoParams defines the contract for bidrequest.imp[i].ext.kadenai.video
type kadenaiVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
