package openrtb_ext

type ImpExtTaboola struct {
	PublisherId     string   `json:"publisherId"`
	PublisherDomain string   `json:"publisherDomain"`
	BidFloor        float64  `json:"bidfloor"`
	TagId           string   `json:"tagid"`
	TagID           string   `json:"tagId"`
	BCat            []string `json:"bcat"`
	BAdv            []string `json:"badv"`
	PageType        string   `json:"pageType"`
	Position        *int     `json:"position"`
}
