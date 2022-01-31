package openrtb_ext

type ExtImpBeachfront struct {
	AppId             string                 `json:"appId"`
	AppIds            ExtImpBeachfrontAppIds `json:"appIds"`
	BidFloor          float64                `json:"bidfloor"`
	VideoResponseType string                 `json:"videoResponseType,omitempty"`
}

type ExtImpBeachfrontAppIds struct {
	Video  string `json:"video,omitempty"`
	Banner string `json:"banner,omitempty"`
}
