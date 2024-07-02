package openrtb_ext

// ExtImpMetaX defines the contract for bidrequest.imp[i].ext.prebid.bidder.metax
type ExtImpMetaX struct {
	PublisherID int `json:"publisherId"`
	Adunit      int `json:"adunit"`
}
