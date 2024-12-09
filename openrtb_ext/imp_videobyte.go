package openrtb_ext

// ExtImpVideoByte defines the contract for bidrequest.imp[i].ext.prebid.bidder.videobyte
type ExtImpVideoByte struct {
	PublisherId string `json:"pubId"`
	PlacementId string `json:"placementId"`
	NetworkId   string `json:"nid"`
}
