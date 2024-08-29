package openrtb_ext

// ExtImpMediaGo defines the contract for bidrequest.imp[i].ext.prebid.bidder.mediago
type ExtImpMediaGo struct {
	Token       string `json:"token"`
	Region      string `json:"region"`
	PlacementId string `json:"placementId"`
}

type ExtMediaGo struct {
	Token  string `json:"token"`
	Region string `json:"region"`
}
