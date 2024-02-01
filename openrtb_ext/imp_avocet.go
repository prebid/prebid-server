package openrtb_ext

// ExtImpAvocet defines the contract for bidrequest.imp[i].ext.prebid.bidder.avocet
type ExtImpAvocet struct {
	Placement     string `json:"placement,omitempty"`
	PlacementCode string `json:"placement_code,omitempty"`
}
