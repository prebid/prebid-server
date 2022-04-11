package openrtb_ext

// ExtImpRTBHouse defines the contract for bidrequest.imp[i].ext.rtbhouse
type ExtImpRTBHouse struct {
	PublisherID    string              `json:"publisherId"`
	Video          rtbHouseVideoParams `json:"video"`
	Region         string              `json:"region"`
	SKADNSupported bool                `json:"skadn_supported"`
	MRAIDSupported bool                `json:"mraid_supported"`
	BidFloor       *float64            `json:"bid_floor,omitempty"`
}

// rtbHouseVideoParams defines the contract for bidrequest.imp[i].ext.rtbhouse.video
type rtbHouseVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}
