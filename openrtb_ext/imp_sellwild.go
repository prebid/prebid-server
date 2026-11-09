package openrtb_ext

// ExtImpSellwild defines the contract for bidrequest.imp[i].ext.prebid.bidder.sellwild
type ExtImpSellwild struct {
	PartnerCode string `json:"partnerCode"`
	Token       string `json:"token"`
}
