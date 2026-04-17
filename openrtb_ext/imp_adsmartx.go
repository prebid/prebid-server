package openrtb_ext

type ImpExtAdsmartx struct {
	BidFloor  float64 `json:"bidfloor,omitempty"`
	TestMode  int     `json:"testMode,omitempty"`
	SspID     string  `json:"sspId,omitempty"`
	SspSiteID string  `json:"siteId,omitempty"`
}
