package openrtb_ext

type ImpExtAlkimi struct {
	Token      string  `json:"token"`
	BidFloor   float64 `json:"bidFloor"`
	Instl      int8    `json:"instl"`
	Exp        int64   `json:"exp"`
	AdUnitCode string  `json:"adUnitCode"`
}
