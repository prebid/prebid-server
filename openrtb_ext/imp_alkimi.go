package openrtb_ext

type ExtImpAlkimi struct {
	Token      string  `json:"token"`
	BidFloor   float64 `json:"bidFloor"`
	Instl      int8    `json:"instl"`
	Exp        int64   `json:"exp"`
	AdUnitCode string  `json:"adUnitCode"`
}
