package openrtb_ext

type ExtImpBeachfront struct {
	AppId    string                 `json:"appId"`
	AppIds   ExtImpBeachfrontAppIds `json:"appIds"`
	BidFloor float64                `json:"bidfloor"`
	NurlVideo bool					`json:"nurlvideo"`
}

type ExtImpBeachfrontAppIds struct {
	Video  string `json:"video"`
	Banner string `json:"banner"`
}
