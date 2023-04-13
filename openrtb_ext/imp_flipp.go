package openrtb_ext

type ImpExtFlipp struct {
	PublisherNameIdentifier string  `json:"publisherNameIdentifier"`
	CreativeType            string  `json:"creativeType"`
	SiteID                  int64   `json:"siteId"`
	ZoneIds                 []int64 `json:"zoneIds,omitempty"`
	UserKey                 string  `json:"userKey,omitempty"`
}
