package openrtb_ext

// ExtImp33across defines the contract for bidrequest.imp[i].ext.prebid.bidder.33across
type ExtImp33across struct {
	SiteId    string `json:"siteId"`
	ZoneId    string `json:"zoneId,omitempty"`
	ProductId string `json:"productId,omitempty"`
}
