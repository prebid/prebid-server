package openrtb_ext

// ExtPublisher defines the contract for ...publisher.ext (found in both bidrequest.site and bidrequest.app)
type ExtPublisher struct {
	Prebid *ExtPublisherPrebid `json:"prebid"`
}

// ExtPublisherPrebid defines the contract for publisher.ext.prebid
type ExtPublisherPrebid struct {
	// parentAccount would define the legal entity (publisher owner or network) that has the direct relationship with the PBS
	// host. As such, the definition depends on the PBS hosting entity.
	ParentAccount *string `json:"parentAccount,omitempty"`
}
