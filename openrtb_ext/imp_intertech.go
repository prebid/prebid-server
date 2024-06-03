package openrtb_ext

type ExtImpIntertech struct {
	PlacementID string `json:"placement_id"`

	// Deprecated: in favor of `PlacementID`
	PageID int64 `json:"page_id"`
	// Deprecated: in favor of `PlacementID`
	ImpID int64 `json:"imp_id"`
}
