package openrtb_ext

type ExtImpScalibur struct {
	PlacementID string   `json:"placementId,omitempty"` // optional; applied to imp.tagid when the imp has no ad-unit-level tagid
	BidFloor    *float64 `json:"bidfloor,omitempty"`    // optional, used as fallback
	BidFloorCur string   `json:"bidfloorcur,omitempty"` // optional, defaults to USD if empty

	// Host optionally fills the {{.Host}} macro in the endpoint template; SSRF-validated.
	Host string `json:"host,omitempty"`
}

type ExtRequestScalibur struct {
	IsDebug int `json:"isDebug,omitempty"`
}
