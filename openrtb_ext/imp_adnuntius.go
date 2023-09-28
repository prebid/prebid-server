package openrtb_ext

type ImpExtAdnunitus struct {
	Auid      string `json:"auId"`
	Network   string `json:"network"`
	NoCookies bool   `json:"noCookies"`
	MaxDeals  int    `json:"maxDeals"`
	BidType   string `json:"bidType,omitempty"`
}
