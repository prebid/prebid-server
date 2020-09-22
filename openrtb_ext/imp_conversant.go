package openrtb_ext

type ExtImpConversant struct {
	SiteID      string   `json:"site_id"`
	Secure      *int8    `json:"secure"`
	TagID       string   `json:"tag_id"`
	Position    *int8    `json:"position"`
	BidFloor    float64  `json:"bidfloor"`
	MIMEs       []string `json:"mimes"`
	API         []int8   `json:"api"`
	Protocols   []int8   `json:"protocols"`
	MaxDuration *int64   `json:"maxduration"`
}
