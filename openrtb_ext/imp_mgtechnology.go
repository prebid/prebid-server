package openrtb_ext

// ExtImpMgTechnology defines the contract for bidrequest.imp[i].ext.prebid.bidder.mgtechnology
type ExtImpMgTechnology struct {
	Token       string `json:"token"`
	Region      string `json:"region"`
	PlacementId string `json:"placementId"`
}

type ExtMgTechnology struct {
	Token  string `json:"token"`
	Region string `json:"region"`
}
