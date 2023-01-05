package openrtb_ext

// ExtImpUnicorn defines the contract for bidrequest.imp[i].ext.prebid.bidder.unicorn
type ExtImpUnicorn struct {
	PlacementID string `json:"placementId,omitempty"`
	PublisherID string `json:"publisherId,omitempty"`
	MediaID     string `json:"mediaId,omitempty"`
	AccountID   int    `json:"accountId,omitempty"`
}
