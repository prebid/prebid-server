package openrtb_ext

// ExtSmartyAds defines the contract for bidrequest.imp[i].ext.smartyads
type ExtSmartyAds struct {
	AccountID string `json:"accountid"`
	SourceID  string `json:"sourceid"`
	Host      string `json:"host"`
}
