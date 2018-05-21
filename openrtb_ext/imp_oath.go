package openrtb_ext

// ExtImpOath defines the contract for bidrequest.imp[i].ext.openx
type ExtImpOath struct {
	PublisherID   string `json:"publisherId"`
	PublisherName string `json:"publisherName"`
	HeaderBidding bool   `json:"headerbidding"`
}
