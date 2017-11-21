package openrtb_ext

// ExtImp defines the contract for bidrequest.imp[i].ext
type ExtImp struct {
	Prebid   *ExtImpPrebid   `json:"prebid"`
	Appnexus *ExtImpAppnexus `json:"appnexus"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	Config *ExtConfig `json:"managedconfig"`
}

// ExtConfig defines the contract for bidrequest.imp[i].ext.prebid.managedconfig
type ExtConfig struct {
	ID string `json:"id"`
}
