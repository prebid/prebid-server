package openrtb_ext

type ImpExtAdnunitus struct {
	Auid      string `json:"auId"`
	NoCookies bool   `json:"noCookies"`
	MaxDeals  int    `json:"maxDeals"`
	Network   string `json:"network"`
	BidType   string `json:"bidType,omitempty"`
	Targeting struct {
		Category            []string            `json:"c,omitempty"`
		Segments            []string            `json:"segments,omitempty"`
		Keywords            []string            `json:"keywords,omitempty"`
		KeyValues           map[string][]string `json:"kv,omitempty"`
		AdUnitMatchingLabel []string            `json:"auml,omitempty"`
	} `json:"targeting,omitempty"`
}
