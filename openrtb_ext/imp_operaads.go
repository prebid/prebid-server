package openrtb_ext

// ExtImpOperaAds defines the contract for bidrequest.imp[i].ext.operaads
type ExtImpOperaAds struct {
	PublisherID    string               `json:"publisherId"`
	Video          operaAdsVideoParams  `json:"video"`
	Region         string               `json:"region"`
	SKADNSupported bool                 `json:"skadn_supported"`
	MRAIDSupported bool                 `json:"mraid_supported"`
	EndpointId     string               `json:"endpointId"`
	PlacementId    operaAdsPlacementIds `json:"placementId"`
}

// operaadsVideoParams defines the contract for bidrequest.imp[i].ext.operaads.video
type operaAdsVideoParams struct {
	Width     int `json:"width,omitempty"`
	Height    int `json:"height,omitempty"`
	Skip      int `json:"skip,omitempty"`
	SkipDelay int `json:"skipdelay,omitempty"`
}

type operaAdsPlacementIds struct {
	Banner string `json:"banner,omitempty"`
	Native string `json:"native,omitempty"`
	Video  string `json:"video,omitempty"`
}
