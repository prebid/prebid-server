package openrtb_ext

// ExtImpEPlanning defines the contract for bidrequest.imp[i].ext.prebid.bidder.eplanning
type ExtImpEPlanning struct {
	ClientID   string `json:"ci"`
	AdUnitCode string `json:"adunit_code"`
	SizeString string
}
