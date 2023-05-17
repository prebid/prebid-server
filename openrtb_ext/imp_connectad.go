package openrtb_ext

type ExtImpConnectAd struct {
	NetworkID int     `json:"networkId"`
	SiteID    int     `json:"siteId"`
	Bidfloor  float64 `json:"bidfloor,omitempty"`
}
