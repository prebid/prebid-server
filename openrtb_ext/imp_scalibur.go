package openrtb_ext

type ExtImpScalibur struct {
	PlacementID string   `json:"placementId,omitempty"` // optional; applied to imp.tagid when the imp has no ad-unit-level tagid
	BidFloor    *float64 `json:"bidfloor,omitempty"`    // optional, used as fallback
	BidFloorCur string   `json:"bidfloorcur,omitempty"` // optional, defaults to USD if empty

	// Host optionally overrides the endpoint host by filling the {{.Host}} macro
	// in the operator-controlled endpoint template (adapters.scalibur.endpoint).
	// When omitted it falls back to the standard Scalibur host, so the default
	// endpoint is preserved unchanged. It is SSRF-validated.
	Host string `json:"host,omitempty"`
}

type ExtRequestScalibur struct {
	IsDebug int `json:"isDebug,omitempty"`
}
