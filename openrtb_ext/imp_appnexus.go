package openrtb_ext

// ExtImpAppnexus defines the contract for bidrequest.imp[i].ext.appnexus
type ExtImpAppnexus struct {
	LegacyPlacementId       int                     `json:"placementId"`
	LegacyInvCode           string                  `json:"invCode"`
	LegacyTrafficSourceCode string                  `json:"trafficSourceCode"`
	PlacementId             int                     `json:"placement_id"`
	InvCode                 string                  `json:"inv_code"`
	Member                  string                  `json:"member"`
	Keywords                []*ExtImpAppnexusKeyVal `json:"keywords"`
	TrafficSourceCode       string                  `json:"traffic_source_code"`
	Reserve                 float64                 `json:"reserve"`
	Position                string                  `json:"position"`
	UsePmtRule              *bool                   `json:"use_pmt_rule"`
}

// ExtImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.appnexus.keywords[i]
type ExtImpAppnexusKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
