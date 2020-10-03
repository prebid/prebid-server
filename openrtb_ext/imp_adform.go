package openrtb_ext

type ExtImpAdform struct {
	MasterTagId string  `json:"mid"`
	PriceType   string  `json:"priceType,omitempty"`
	KeyValues   string  `json:"mkv,omitempty"`
	KeyWords    string  `json:"mkw,omitempty"`
	CDims       string  `json:"cdims,omitempty"`
	MinPrice    float64 `json:"minp,omitempty"`
	Url         string  `json:"url,omitempty"`
}
