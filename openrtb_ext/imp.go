package openrtb_ext

// ExtImp defines the contract for bidrequest.imp[i].ext
type ExtImp struct {
	Prebid ExtBidPrebid `json:"prebid"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid
type ExtImpPrebid struct {
	Config ExtConfig `json:"managedconfig"`
}

// ExtImpPrebid defines the contract for bidrequest.imp[i].ext.prebid.managedconfig
type ExtConfig struct {
	ID string `json:"id"`
}
