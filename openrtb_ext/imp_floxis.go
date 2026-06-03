package openrtb_ext

// ExtImpFloxis defines the contract for bidrequest.imp[i].ext.prebid.bidder.floxis.
type ExtImpFloxis struct {
	Seat   string `json:"seat"`
	Region string `json:"region,omitempty"`
}
