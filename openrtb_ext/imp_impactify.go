package openrtb_ext

// ExtImpImpactify defines the contract for bidrequest.imp[i].ext.prebid.bidder.impactify
type ExtImpImpactify struct {
	AppID  string `json:"appId"`
	Format string `json:"format"`
	Style  string `json:"style"`
}
