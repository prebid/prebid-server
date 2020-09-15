package openrtb_ext

// ExtSmartyAds defines the contract for bidrequest.imp[i].ext.smartyads
type ExtSmartyAds struct {
	AccountID string `json:"accountid"`
	SourceId  string `json:"sourceid"`
	Host      string `json:"host"`
}
