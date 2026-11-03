package openrtb_ext

type ImpExtNextMillennium struct {
	GroupID     string   `json:"group_id,omitempty"`
	PlacementID string   `json:"placement_id"`
	AdSlots     []string `json:"adSlots,omitempty"`
	AllowedAds  []string `json:"allowedAds,omitempty"`
}
