package openrtb_ext

// ExtImpTradPlus defines the contract for bidrequest.imp[i].ext.prebid.bidder.tradplus
type ExtImpTradPlus struct {
	AccountID string `json:"accountId"`
	ZoneID    string `json:"zoneId"`
}
