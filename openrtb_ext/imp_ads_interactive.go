package openrtb_ext

type ImpExtAdsInteractive struct {
	PlacementID string `json:"placementId"`
	EndpointID  string `json:"endpointId"`
	AdUnit      string `json:"adUnit"` // Deprecated, use placementId or endpointId instead
}
