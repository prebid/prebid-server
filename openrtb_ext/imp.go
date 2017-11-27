package openrtb_ext

// ExtImp defines the contract for bidrequest.imp[i].ext
type ExtImp struct {
	Prebid   *ExtImpPrebid   `json:"prebid"`
	Appnexus *ExtImpAppnexus `json:"appnexus"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	StoredRequest *ExtConfig `json:"storedrequest"`
}

// ExtConfig defines the contract for bidrequest.imp[i].ext.prebid.storedrequest
type ExtConfig struct {
	ID string `json:"id"`
}
