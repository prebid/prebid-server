package openrtb_ext

// ImpExtCamino defines the contract for bidrequest.imp[i].ext.prebid.bidder.camino
type ImpExtCamino struct {
	PlacementID string `json:"placementId"`
	PublisherID string `json:"publisherId,omitempty"`
}
