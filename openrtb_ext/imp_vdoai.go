package openrtb_ext

import "encoding/json"

type ImpExtVdoai struct {
	Host        string          `json:"host,omitempty"`
	AdUnitId    json.RawMessage `json:"adUnitId"`
	AdUnitType  string          `json:"adUnitType"`
	PublisherId string          `json:"publisherId,omitempty"`
	BidFloor    float64         `json:"bidfloor,omitempty"`
	Custom1     string          `json:"custom1,omitempty"`
	Custom2     string          `json:"custom2,omitempty"`
	Custom3     string          `json:"custom3,omitempty"`
	Custom4     string          `json:"custom4,omitempty"`
	Custom5     string          `json:"custom5,omitempty"`
}
