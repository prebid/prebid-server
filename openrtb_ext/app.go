package openrtb_ext

// ExtApp defines the contract for bidrequest.app.ext
type ExtApp struct {
	Prebid ExtAppPrebid `json:"prebid"`
}

// ExtAppPrebid further defines the contract for bidrequest.app.ext.prebid.
// We are only enforcing that these two properties be strings if they are provided.
// They are optional with no current constraints on value, so we don't need a custom
// UnmarshalJSON() method at this time.
type ExtAppPrebid struct {
	Source  string `json:"source"`
	Version string `json:"version"`
}
