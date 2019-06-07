package openrtb_ext

type ExtImpAdform struct {
	MasterTagId string `json:"mid"`
	PriceType   string `json:"priceType,omitempty"`
	KeyValues   string `json:"mkv,omitempty"`
	Keywords    string `json:"mkw,omitempty"`
}
