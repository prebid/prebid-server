package openrtb_ext

// ImpExtNativery defines the contract for bidrequest.imp[i].ext.prebid.bidder.nativery
// ref to json schema in static/bidder-params/nativery

type ImpExtNativery struct {
	PlacementID string `json:"placement_id"`
}