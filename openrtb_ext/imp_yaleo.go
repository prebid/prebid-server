package openrtb_ext

// ImpExtYaleo defines the contract for bidrequest.imp[i].ext.prebid.bidder.yaleo
type ImpExtYaleo struct {
	PlacementID string   `json:"placementId"`
	MemberID    *int     `json:"memberId,omitempty"`
	MaxCPM      *float64 `json:"maxCpm,omitempty"`
}
