package openrtb_ext

// ExtImpUcfunnel defines the contract for bidrequest.imp[i].ext.prebid.bidder.ucfunnel
type ExtImpUcfunnel struct {
	AdUnitId  string `json:"adunitid"`
	PartnerId string `json:"partnerid"`
}
