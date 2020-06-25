package openrtb_ext

import "encoding/json"

type ExtImpTelaria struct {
	AdCode   string          `json:"adCode,omitempty"`
	SeatCode string          `json:"seatCode"`
	Extra    json.RawMessage `json:"extra,omitempty"`
}
