package openrtb_ext

import "encoding/json"

// ExtImpGoadserver defines the contract for bidrequest.imp[i].ext.prebid.bidder.goadserver
type ExtImpGoadserver struct {
	Token string          `json:"token"`
	Floor float64         `json:"floor,omitempty"`
	SubID json.RawMessage `json:"subid,omitempty"`
}
