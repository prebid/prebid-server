package openrtb_ext

// ExtImpGamoshi defines the contract for bidrequest.imp[i].ext.prebid.bidder.gamoshi
type ExtImpGamoshi struct {
	SupplyPartnerId  string `json:"supplyPartnerId"`
	FavoredMediaType string `json:"favoredMediaType"`
}
