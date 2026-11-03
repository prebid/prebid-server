package openrtb_ext

// ExtImpOms defines the contract for bidrequest.imp[i].ext.prebid.bidder.oms
type ExtImpOms struct {
	Pid         string `json:"pid"`
	PublisherID int    `json:"publisherId"`
}
