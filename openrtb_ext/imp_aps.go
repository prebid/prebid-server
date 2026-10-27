package openrtb_ext

// ExtImpAps defines bidrequest.imp[i].ext.prebid.bidder.aps.
type ExtImpAps struct {
	AccountID string `json:"accountID"`
	Region    string `json:"region,omitempty"`
}
