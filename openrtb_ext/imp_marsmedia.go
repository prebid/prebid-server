package openrtb_ext

import "encoding/json"

// ExtImpMarsmedia defines the contract for bidrequest.imp[i].ext.marsmedia
type ExtImpMarsmedia struct {
	ZoneID json.Number `json:"zoneId"`
}
