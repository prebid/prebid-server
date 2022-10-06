package openrtb_ext

// ExtApp defines the contract for bidrequest.app.ext
type ExtApp struct {
	Prebid ExtAppPrebid `json:"prebid"`
}

// ExtAppPrebid further defines the contract for bidrequest.app.ext.prebid.
type ExtAppPrebid struct {
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}
