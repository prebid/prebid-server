package openrtb_ext

type ExtImpTappx struct {
	Host     string   `json:"host,omitempty"` //DEPRECATED
	TappxKey string   `json:"tappxkey"`
	Endpoint string   `json:"endpoint"`
	BidFloor float64  `json:"bidfloor,omitempty"`
	Mktag    string   `json:"mktag,omitempty"`
	Bcid     []string `json:"bcid,omitempty"`
	Bcrid    []string `json:"bcrid,omitempty"`
}
