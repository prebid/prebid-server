package openrtb_ext

// ExtImpAppnexus defines the contract for bidrequest.imp[i].ext.appnexus
type ExtImpAppnexus struct {
	PlacementId       int                     `json:"placementId"`
	InvCode           string                  `json:"invCode"`
	Member            string                  `json:"member"`
	Keywords          []*ExtImpAppnexusKeyVal `json:"keywords"`
	TrafficSourceCode string                  `json:"trafficSourceCode"`
	Reserve           float64                 `json:"reserve"`
	Position          string                  `json:"position"`
}

// ExtImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.appnexus.keywords[i]
type ExtImpAppnexusKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
