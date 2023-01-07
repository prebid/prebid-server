package openrtb_ext

// ExtSmartyAds defines the contract for bidrequest.imp[i].ext.prebid.bidder.smartyads
type ExtSmartyAds struct {
	AccountID string `json:"accountid"`
	SourceID  string `json:"sourceid"`
	Host      string `json:"host"`
}
