package openrtb_ext

type ExtImpConnectAd struct {
	NetworkId int     `json:"networkId"`
	SiteId    int     `json:"siteId"`
	Bidfloor  float64 `json:"bidfloor,omitempty"`
}
