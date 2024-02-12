package openrtb_ext

type ExtImpYandex struct {
	/*
		Possible formats
			- `R-I-123456-2`
			- `R-123456-1`
			- `123456-789`
	*/
	PlacementID string `json:"placement_id"`

	// Deprecated: in favor of `PlacementID`
	PageID int64 `json:"page_id"`
	// Deprecated: in favor of `PlacementID`
	ImpID int64 `json:"imp_id"`
}
