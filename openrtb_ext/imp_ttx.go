package openrtb_ext

// ExtImpTtx defines the contract for bidrequest.imp[i].ext.ttx
type ExtImpTtx struct {
	SiteId    string `json:"siteId"`
	ZoneId    string `json:"zoneId,omitempty"`
	ProductId string `json:"productId,omitempty"`
}
