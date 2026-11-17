package openrtb_ext

// ExtImpTriplelift defines the contract for bidrequest.imp[i].ext.prebid.bidder.triplelift
type ExtImpTriplelift struct {
	InvCode     string   `json:"inventoryCode"`
	Floor       *float64 `json:"floor"`
	ParentID    *string  `json:"parentId,omitempty"`
	PublisherID *string  `json:"publisherId,omitempty"`
}
