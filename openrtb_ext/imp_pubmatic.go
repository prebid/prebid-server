package openrtb_ext

import "encoding/json"

// ExtImpPubmatic defines the contract for bidrequest.imp[i].ext.pubmatic
// PublisherId and adSlot are mandatory parameters, others are optional parameters
// Keywords, Kadfloor are bid specific parameters,
// other parameters Lat,Lon, Yob, Kadpageurl, Gender, Yob, WrapExt needs to sent once per bid  request

type ExtImpPubmatic struct {
	PublisherId string            `json:"publisherId"`
	AdSlot      string            `json:"adSlot"`
	Lat         float64           `json:"lat,omitempty"`
	Lon         float64           `json:"lon,omitempty"`
	Yob         int               `json:"yob,omitempty"`
	Kadpageurl  string            `json:"kadpageurl,omitempty"`
	Gender      string            `json:"gender,omitempty"`
	Kadfloor    float64           `json:"kadfloor,omitempty"`
	WrapExt     json.RawMessage   `json:"wrapper,omitempty"`
	Keywords    map[string]string `json:"keywords,omitempty"`
}
