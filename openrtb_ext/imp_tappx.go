package openrtb_ext

type ExtImpTappx struct {
	Host     string  `json:"host"`
	TappxKey string  `json:"tappxkey"`
	Endpoint string  `json:"endpoint"`
	BidFloor float64 `json:"bidfloor,omitempty"`
}
