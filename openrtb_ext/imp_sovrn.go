package openrtb_ext

import "encoding/json"

type ExtImpSovrn struct {
	TagId      string          `json:"tagId,omitempty"`
	Tagid      string          `json:"tagid,omitempty"`
	BidFloor   json.RawMessage `json:"bidfloor,omitempty"`
	AdUnitCode string          `json:"adunitcode,omitempty"`
}
