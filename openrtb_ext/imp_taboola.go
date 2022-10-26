package openrtb_ext

type ImpExtTaboola struct {
	PublisherId string   `json:"publisherId"`
	BidFloor    float64  `json:"bidfloor"`
	TagId       string   `json:"tagid"`
	BCat        []string `json:"bcat"`
	BAdv        []string `json:"badv"`
}
