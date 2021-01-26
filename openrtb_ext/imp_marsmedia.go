package openrtb_ext

// ExtImpMarsmedia defines the contract for bidrequest.imp[i].ext.marsmedia
type ExtImpMarsmedia struct {
	ZoneID   string  `json:"zone"`
	BidFloor float64 `json:"bidFloor,omitempty"`
}
