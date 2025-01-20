package openrtb_ext

type ExtImpConnatix struct {
	PlacementId string `json:"placementId"`
	DeclaredViewabilityPercentage     float64     `json:"declaredViewabilityPercentage"`
    DetectedViewabilityPercentage     float64     `json:"detectedViewabilityPercentage"`
}
