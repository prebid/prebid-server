package openrtb_ext

// ExtImpMediaGoTechnology defines the contract for bidrequest.imp[i].ext.prebid.bidder.mediagoTechnology
type ExtImpMediaGoTechnology struct {
	Token       string `json:"token"`
	Region      string `json:"region"`
	PlacementId string `json:"placementId"`
}

type ExtMediaGoTechnology struct {
	Token  string `json:"token"`
	Region string `json:"region"`
}
