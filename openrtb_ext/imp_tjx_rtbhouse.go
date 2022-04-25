package openrtb_ext

// ExtImpTJXRTBHouse defines the contract for bidrequest.imp[i].ext.rtbhouse
type ExtImpTJXRTBHouse struct {
	PublisherID    string                 `json:"publisherId"`
	Video          tjxRtbHouseVideoParams `json:"video"`
	Region         string                 `json:"region"`
	SKADNSupported bool                   `json:"skadn_supported"`
	MRAIDSupported bool                   `json:"mraid_supported"`
	BidFloor       *float64               `json:"bid_floor,omitempty"`
}

// rtbHouseVideoParams defines the contract for bidrequest.imp[i].ext.rtbhouse.video
type tjxRtbHouseVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
