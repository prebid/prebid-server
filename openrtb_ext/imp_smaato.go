package openrtb_ext

// ExtImpSmaato defines the contract for bidrequest.imp[i].ext.smaato
// PublisherId and AdSpaceId are mandatory parameters for non adpod (long-form video) requests, others are optional parameters
// PublisherId and AdBreakId are mandatory parameters for adpod (long-form video) requests, others are optional parameters
// AdSpaceId is identifier for specific ad placement or ad tag
// AdBreakId is identifier for specific ad placement or ad tag
type ExtImpSmaato struct {
	PublisherID string `json:"publisherId"`
	AdSpaceID   string `json:"adspaceId"`
	AdBreakID   string `json:"adbreakId"`
}
