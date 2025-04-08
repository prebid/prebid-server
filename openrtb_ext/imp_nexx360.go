package openrtb_ext

// ExtNexx360 defines the contract for bidrequest.imp[i].ext.prebid.bidder.nexx360
type ExtImpNexx360 struct {
	TagId     string `json:"tagId,omitempty"`
	Placement string `json:"placement,omitempty"` // Placement ID
}
