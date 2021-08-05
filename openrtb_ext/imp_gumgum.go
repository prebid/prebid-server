package openrtb_ext

// ExtImpGumGum defines the contract for bidrequest.imp[i].ext.gumgum
// Either Zone or PubId must be present, others are optional parameters
type ExtImpGumGum struct {
	Zone   string  `json:"zone,omitempty"`
	PubID  float64 `json:"pubId,omitempty"`
	IrisID string  `json:"irisid,omitempty"`
	Slot   float64 `json:"slot,omitempty"`
}

// ExtImpGumGumVideo defines the contract for bidresponse.seatbid.bid[i].ext.gumgum.video
type ExtImpGumGumVideo struct {
	IrisID string `json:"irisid,omitempty"`
}

// ExtImpGumGumBanner defines the contract for bidresponse.seatbid.bid[i].ext.gumgum.banner
type ExtImpGumGumBanner struct {
	Si   float64 `json:"si,omitempty"`
	MaxW float64 `json:"maxw,omitempty"`
	MaxH float64 `json:"maxh,omitempty"`
}
