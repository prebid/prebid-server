package openrtb_ext

// ExtImpAnzuPrebid defines the contract for bidrequest.imp[i].ext.prebid.bidder.anzuprebid.
type ExtImpAnzuPrebid struct {
	Zone        string `json:"zone"`
	QueryString string `json:"queryString"`
}
