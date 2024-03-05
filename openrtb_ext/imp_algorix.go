package openrtb_ext

// ExtImpAlgoriX defines the contract for bidrequest.imp[i].ext.prebid.bidder.algorix
type ExtImpAlgorix struct {
	Sid         string `json:"sid"`
	Token       string `json:"token"`
	PlacementId string `json:"placementId"`
	AppId       string `json:"appId"`
	Region      string `json:"region"`
}
