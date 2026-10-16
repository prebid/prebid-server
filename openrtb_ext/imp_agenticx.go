package openrtb_ext

type ImpExtAgenticx struct {
	BidFloor  float64 `json:"bidFloor,omitempty"`
	TestMode  int     `json:"testMode,omitempty"`
	SspID     string  `json:"sspId,omitempty"`
	SspSiteID string  `json:"siteId,omitempty"`
}
