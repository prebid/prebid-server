package openrtb_ext

import "encoding/json"

type ExtImpTelaria struct {
	AdCode       string          `json:"adCode,omitempty"`
	SeatCode     string          `json:"seatCode"`
	CustomParams json.RawMessage `json:"customParams"`
}
