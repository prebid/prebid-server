package openrtb_ext

// ExtApp defines the contract for bidrequest.app.ext
type ExtApp struct {
	Prebid ExtBidPrebid `json:"prebid"`
}

// ExtApp defines the contract for bidrequest.app.ext.prebid
type ExtAppPrebid struct {
	AppSource  string `json:"source"`
	AppVersion string `json:"version"`
}
