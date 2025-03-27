package openrtb_ext

type ImpExtReadpeak struct {
	PublisherId string  `json:"publisherId"`
	SiteId      string  `json:"siteId"`
	Bidfloor    float64 `json:"bidfloor"`
	TagId       string  `json:"tagId"`
}
