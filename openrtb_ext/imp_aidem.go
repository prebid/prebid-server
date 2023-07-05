package openrtb_ext

type ImpExtFoo struct {
	SiteID      string `json:"siteId"`
	PublisherID string `json:"publisherId"`
	PlacementID string `json:"placementId"`
	RateLimit   string `json:"rateLimit"`
}
