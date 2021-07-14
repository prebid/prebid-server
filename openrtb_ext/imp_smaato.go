package openrtb_ext

// ExtImpSmaato defines the contract for bidrequest.imp[i].ext.smaato
// PublisherId and AdSpaceId are mandatory parameters, others are optional parameters
// AdSpaceId is identifier for specific ad placement or ad tag
type ExtImpSmaato struct {
	PublisherID string `json:"publisherId"`
	AdSpaceID   string `json:"adspaceId"`
}
