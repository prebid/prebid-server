package openrtb_ext

type ImpExtKargo struct {
	PlacementId string `json:"placementId"`
	AdSlotID    string `json:"adSlotID"` // Deprecated - Use `placementId`
}
